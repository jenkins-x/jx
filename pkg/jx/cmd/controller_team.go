package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ControllerTeamOptions are the flags for the commands
type ControllerTeamOptions struct {
	ControllerOptions
	InstallOptions

	GitRepositoryOptions gits.GitRepositoryOptions
}

// NewCmdControllerTeam creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdControllerTeam(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &ControllerTeamOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
		InstallOptions: CreateInstallOptions(f, in, out, errOut),
	}

	cmd := &cobra.Command{
		Use:   "team",
		Short: "Runs the team controller",
		Run: func(cmd *cobra.Command, args []string) {
			options.ControllerOptions.Cmd = cmd
			options.ControllerOptions.Args = args
			err := options.Run()
			CheckErr(err)
		},
		Aliases: []string{"team"},
	}

	options.ControllerOptions.addCommonFlags(cmd)
	options.InstallOptions.addInstallFlags(cmd, true)

	return cmd
}

// Run implements this command
func (o *ControllerTeamOptions) Run() error {
	co := &o.ControllerOptions
	err := co.registerTeamCRD()
	if err != nil {
		return err
	}

	jxClient, devNs, err := co.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	log.Infof("Using the admin namespace %s\n", devNs)

	client, _, err := co.KubeClient()
	if err != nil {
		return err
	}

	// lets default the team settings based on the current team settings
	settings, err := co.TeamSettings()
	if err != nil {
		return errors.Wrapf(err, "Failed to get TeamSettings")
	}
	if settings == nil {
		return fmt.Errorf("No TeamSettings found!")
	}
	if settings.HelmTemplate {
		o.InstallOptions.InitOptions.Flags.NoTiller = true
	} else if settings.NoTiller {
		o.InstallOptions.InitOptions.Flags.RemoteTiller = false
	} else if settings.HelmBinary == "helm3" {
		o.InstallOptions.InitOptions.Flags.Helm3 = true
	}
	if settings.PromotionEngine == v1.PromotionEngineProw {
		o.InstallOptions.Flags.Prow = true
	}

	log.Infof("Watching for teams in all namespaces\n")

	stop := make(chan struct{})

	_, teamController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo meta_v1.ListOptions) (runtime.Object, error) {
				return jxClient.JenkinsV1().Teams("").List(lo)
			},
			WatchFunc: func(lo meta_v1.ListOptions) (watch.Interface, error) {
				return jxClient.JenkinsV1().Teams("").Watch(lo)
			},
		},
		&v1.Team{},
		time.Minute*30,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onTeamChange(obj, client, jxClient, devNs)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onTeamChange(newObj, client, jxClient, devNs)
			},
			DeleteFunc: func(obj interface{}) {
				// do nothing, already handled by 'jx delete team'
			},
		},
	)

	go teamController.Run(stop)

	// Wait forever
	select {}
}

func (o *ControllerTeamOptions) onTeamChange(obj interface{}, kubeClient kubernetes.Interface, jxClient versioned.Interface, adminNs string) {
	team, ok := obj.(*v1.Team)
	if !ok {
		log.Infof("Object is not a Team %#v\n", obj)
		return
	}

	log.Infof("Adding / Updating Team %s, Namespace %s, Status '%s'\n", util.ColorInfo(team.Name), util.ColorInfo(team.Namespace), util.ColorInfo(team.Status.ProvisionStatus))

	if v1.TeamProvisionStatusNone == team.Status.ProvisionStatus {
		// update first
		err := o.ControllerOptions.ModifyTeam(team.Name, func(team *v1.Team) error {
			team.Status.ProvisionStatus = v1.TeamProvisionStatusPending
			team.Status.Message = "Installing resources"
			return nil
		})
		if err != nil {
			log.Errorf("Unable to update team %s to %s - %s", util.ColorInfo(team.Name), v1.TeamProvisionStatusPending, err)
			return
		}

		// ensure that the namespace exists
		err = kube.EnsureNamespaceCreated(kubeClient, team.Name, nil, nil)
		if err != nil {
			log.Errorf("Unable to create namespace %s: %s", util.ColorInfo(team.Name), err)
			return
		}

		// lets default the login/pwd for Jenkins from the admin cluster
		o.InstallOptions.AdminSecretsService.Flags.DefaultAdminPassword, err = o.ControllerOptions.getDefaultAdminPassword(adminNs)
		if err != nil {
			log.Warnf("Failed to load the default admin password from namespace %s: %s", adminNs, err)
		}

		if o.InstallOptions.CreateEnvOptions.HelmValuesConfig.ExposeController == nil {
			o.InstallOptions.CreateEnvOptions.HelmValuesConfig.ExposeController = &config.ExposeController{}
		}
		ec := o.InstallOptions.CreateEnvOptions.HelmValuesConfig.ExposeController
		// lets load the exposecontroller configuration
		ingressConfig, err := kube.GetIngressConfig(kubeClient, adminNs)
		if err != nil {
			log.Errorf("Failed to load the IngressConfig in namespace %s: %s", adminNs, err)
			return
		}
		ec.Config.Domain = ingressConfig.Domain
		ec.Config.Exposer = ingressConfig.Exposer
		if ingressConfig.TLS {
			ec.Config.HTTP = "false"
			ec.Config.TLSAcme = "true"
		} else {
			ec.Config.HTTP = "true"
			ec.Config.TLSAcme = "false"
		}

		o.InstallOptions.BatchMode = true
		o.InstallOptions.InitOptions.Flags.SkipIngress = true

		adminTeamSettings, _ := o.ControllerOptions.TeamSettings()

		// TODO lets load this from the current team
		provider := ""
		if adminTeamSettings != nil {
			provider = adminTeamSettings.KubeProvider
		}
		if provider == "" {
			log.Warnf("No kube provider specified on admin team settings %s\n", adminNs)
			provider = "gke"
		}
		o.InstallOptions.Flags.Provider = provider

		//o.InstallOptions.Flags.NoDefaultEnvironments = true
		o.InstallOptions.Flags.Namespace = team.Name
		o.InstallOptions.Flags.DefaultEnvironmentPrefix = team.Name
		o.InstallOptions.CommonOptions.InstallDependencies = true

		// call jx install
		installOpts := &o.InstallOptions

		err = installOpts.Run()
		if err != nil {
			log.Errorf("Unable to install jx for team %s: %s", util.ColorInfo(team.Name), err)
			err = o.ControllerOptions.ModifyTeam(team.Name, func(team *v1.Team) error {
				team.Status.ProvisionStatus = v1.TeamProvisionStatusError
				team.Status.Message = err.Error()
				return nil
			})
			if err != nil {
				log.Errorf("Unable to update team %s to %s - %s", util.ColorInfo(team.Name), v1.TeamProvisionStatusError, err)
				return
			}
			return
		}

		if adminTeamSettings != nil {
			callback := func(env *v1.Environment) error {
				env.Spec.TeamSettings.BuildPackRef = adminTeamSettings.BuildPackRef
				env.Spec.TeamSettings.BuildPackURL = adminTeamSettings.BuildPackURL
				return nil
			}
			err = o.ControllerOptions.modifyDevEnvironment(jxClient, team.Name, callback)
			if err != nil {
				log.Errorf("Failed to update team settings in namespace %s: %s\n", team.Name, err)
			}
		}

		err = o.ControllerOptions.ModifyTeam(team.Name, func(team *v1.Team) error {
			team.Status.ProvisionStatus = v1.TeamProvisionStatusComplete
			team.Status.Message = "Installation complete"
			return nil
		})
		if err != nil {
			log.Errorf("Unable to update team %s to %s - %s", util.ColorInfo(team.Name), v1.TeamProvisionStatusComplete, err)
			return
		}
	}
}
