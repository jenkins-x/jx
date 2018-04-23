package cmd

import (
	"io"

	"github.com/spf13/cobra"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"

	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	defaultAnchoreNamespace   = "anchore"
	defaultAnchoreReleaseName = "anchore"
	defaultAnchoreVersion     = "0.1.4"
	defaultAnchorePassword    = "anchore"
	defaultAnchoreConfigDir   = "/anchore_service_dir"
	anchoreServiceName        = "anchore-anchore-engine"
)

var (
	createAddonAnchoreLong = templates.LongDesc(`
		Creates the anchore addon for serverless on kubernetes
`)

	createAddonAnchoreExample = templates.Examples(`
		# Create the anchore addon 
		jx create addon anchore

		# Create the anchore addon in a custom namespace
		jx create addon anchore -n mynamespace
	`)
)

// CreateAddonAnchoreOptions the options for the create spring command
type CreateAddonAnchoreOptions struct {
	CreateAddonOptions

	Chart     string
	Password  string
	ConfigDir string
}

// NewCmdCreateAddonAnchore creates a command object for the "create" command
func NewCmdCreateAddonAnchore(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonAnchoreOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "anchore",
		Short:   "Create the Anchore addon for verifying container images",
		Aliases: []string{"env"},
		Long:    createAddonAnchoreLong,
		Example: createAddonAnchoreExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultAnchoreNamespace, defaultAnchoreReleaseName)

	cmd.Flags().StringVarP(&options.Version, "version", "v", defaultAnchoreVersion, "The version of the Anchore chart to use")
	cmd.Flags().StringVarP(&options.Password, "password", "p", defaultAnchorePassword, "The default password to use for Anchore")
	cmd.Flags().StringVarP(&options.ConfigDir, "config-dir", "d", defaultAnchoreConfigDir, "The config directory to use")
	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartAnchore, "The name of the chart to use")
	return cmd
}

// Run implements the command
func (o *CreateAddonAnchoreOptions) Run() error {

	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	_, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	// todo get current team default admin password?

	devNamespace, _, err := kube.GetDevNamespace(o.kubeClient, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	log.Infof("found dev namespace %s\n", devNamespace)
	//
	//values := []string{"globalConfig.users.admin.password=" + o.Password, "globalConfig.configDir=/anchore_service_dir"}
	//err = o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values)
	//if err != nil {
	//	return fmt.Errorf("error with anchore deployment error: %v", err)
	//}

	log.Info("waiting for anchore deployment to be ready, this may take a few minutes\n")

	err = kube.WaitForDeploymentToBeReady(o.kubeClient, "anchore-anchore-engine-core", o.Namespace, 10*time.Minute)
	if err != nil {
		return err
	}
	svc, err := o.kubeClient.CoreV1().Services(o.Namespace).Get(anchoreServiceName, meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Service %s: %v", anchoreServiceName, err)
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}

	if svc.Annotations["fabric8.io/expose"] == "" {
		svc.Annotations["fabric8.io/expose"] = "true"
		svc, err = o.kubeClient.CoreV1().Services(o.Namespace).Update(svc)
		if err != nil {
			return fmt.Errorf("failed to update service %s/%s", o.Namespace, anchoreServiceName)
		}
	}

	exposecontrollerConfig, err := kube.GetTeamExposecontrollerConfig(o.kubeClient, devNamespace)
	if err != nil {
		return fmt.Errorf("error getting existing team exposecontroller config from namespace %s.  error: %v", o.currentNamespace, err)
	}
	// run exposecontroller using existing team config
	exValues := []string{"config.exposer=" + exposecontrollerConfig["exposer"], "config.domain=" + exposecontrollerConfig["domain"], "config.http=" + exposecontrollerConfig["http"], "config.tls-acme=" + exposecontrollerConfig["tls-acme"]}
	err = o.installChart("ex", "chartmuseum/exposecontroller", "2.3.56", o.Namespace, true, exValues)
	if err != nil {
		return fmt.Errorf("error with exposecontroller deployment error: %v", err)
	}
	//	err = kube.WaitForJobToComplete(o.kubeClient, o.Namespace, "exposecontroller", 1*time.Minute)
	if err != nil {
		return fmt.Errorf("error waiting for exposecontroller job to complete: %v", err)
	}
	err = kube.DeleteJob(o.kubeClient, o.Namespace, "exposecontroller")
	if err != nil {
		return err
	}
	svc, err = o.kubeClient.CoreV1().Services(o.Namespace).Get(anchoreServiceName, meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Service %s: %v", anchoreServiceName, err)
	}
	ing := kube.GetServiceURL(svc)

	log.Infof("got external URL %s\n", ing)

	tokenOptions := CreateTokenAddonOptions{
		Password:  o.Password,
		Username:  "admin",
		Namespace: o.Namespace,
		ServerFlags: ServerFlags{
			ServerURL:  ing,
			ServerName: AnchoreDeploymentName,
		},
		Kind: AnchoreDeploymentName,
		CreateOptions: CreateOptions{
			CommonOptions: o.CommonOptions,
		},
	}
	err = tokenOptions.Run()
	if err != nil {
		return fmt.Errorf("error creating addonAuth.yaml error: %v", err)
	}
	// todo create secret in the dev team namespace with anchore admin password to use in pipeline step

	return nil
}
