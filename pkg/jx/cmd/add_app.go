package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddAppOptions the options for the create spring command
type AddAppOptions struct {
	AddOptions

	GitOps bool
	DevEnv *v1.Environment

	Repo     string
	Username string
	Password string
	Alias    string

	// for testing
	FakePullRequests CreateEnvPullRequestFn

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback ConfigureGitFolderFn

	Namespace   string
	Version     string
	ReleaseName string
	SetValues   string
	ValueFiles  []string
	HelmUpdate  bool
}

// NewCmdAddApp creates a command object for the "create" command
func NewCmdAddApp(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &AddAppOptions{
		AddOptions: AddOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "app",
		Short: "Adds an app",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addFlags(cmd, kube.DefaultNamespace, "", "")
	return cmd
}

func (o *AddAppOptions) addFlags(cmd *cobra.Command, defaultNamespace string, defaultOptionRelease string, defaultVersion string) {

	// We're going to need to know whether the team is using GitOps for the dev env or not,
	// and also access the team settings, so load those
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		if o.Verbose {
			log.Errorf("Error loading team settings. %v\n", err)
		}
		o.GitOps = false
		o.DevEnv = &v1.Environment{}
	} else {
		devEnv, err := kube.GetDevEnvironment(jxClient, ns)
		if err != nil {
			log.Errorf("Error loading team settings. %v\n", err)
			o.GitOps = false
			o.DevEnv = &v1.Environment{}
		} else {
			o.DevEnv = devEnv
			if o.DevEnv.Spec.Source.URL != "" {
				o.GitOps = true
			}
		}
	}

	// Common flags
	cmd.Flags().StringVarP(&o.SetValues, "set", "s", "",
		"The chart set values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	cmd.Flags().StringVarP(&o.Version, "version", "v", defaultVersion,
		"The chart version to install")
	cmd.Flags().StringVarP(&o.Repo, "repository", "", o.DevEnv.Spec.TeamSettings.AppsRepository,
		"The repository from which the chart should be installed")
	cmd.Flags().StringVarP(&o.Version, "username", "", "",
		"The username for the chart repository")
	cmd.Flags().StringVarP(&o.Version, "password", "", "",
		"The password for the chart repository")
	cmd.Flags().BoolVarP(&o.BatchMode, optionBatchMode, "b", false, "In batch mode the command never prompts for user input")
	cmd.Flags().BoolVarP(&o.Verbose, optionVerbose, "", false, "Enable verbose logging")
	if o.GitOps {
		// GitOps specific flags go here
		cmd.Flags().StringVarP(&o.Alias, "alias", "", "", "An alias to use for the app")
	} else {
		// Non GitOps specific flags go here
		cmd.Flags().StringVarP(&o.ReleaseName, optionRelease, "r", defaultOptionRelease, "The chart release name")
		cmd.Flags().BoolVarP(&o.HelmUpdate, "helm-update", "", true, "Should we run helm update first to ensure we use the latest version")
		cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", defaultNamespace, "The Namespace to install into")
		cmd.Flags().StringArrayVarP(&o.ValueFiles, "values", "f", []string{}, "List of locations for values files, can be local files or URLs")
	}

}

// Run implements this command
func (o *AddAppOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return o.Cmd.Help()
	}
	if len(args) > 1 {
		return o.Cmd.Help()
	}
	for _, arg := range args {
		if o.GitOps {
			err := o.createPR(arg)
			if err != nil {
				return err
			}
		} else {
			err := o.installApp(arg)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func (o *AddAppOptions) createPR(app string) error {
	version := o.Version
	if version == "" {
		var err error
		version, err = helm.GetLatestVersion(app, o.Repo, o.Helm())
		if err != nil {
			return err
		}
		if o.Verbose {
			log.Infof("No version specified so using latest version which is %s\n", util.ColorInfo(version))
		}
	}
	modifyRequirementsFn := func(requirements *helm.Requirements) error {
		// See if the app already exists in requirements
		found := false
		for _, d := range requirements.Dependencies {
			if d.Name == app && d.Repository == o.Repo && d.Alias == o.Alias {
				// App found
				log.Infof("App %s already installed.\n", util.ColorWarning(app))
				if version != d.Version {
					log.Infof("To upgrade the app use %s or %s\n",
						util.ColorInfo("jx upgrade app <app>"),
						util.ColorInfo("jx upgrade apps --all"))
				}
				found = true
				break
			}
		}
		// If app not found, add it
		if !found {
			requirements.Dependencies = append(requirements.Dependencies, &helm.Dependency{
				Alias:      o.Alias,
				Repository: o.Repo,
				Name:       app,
				Version:    version,
			})
		}
		return nil
	}
	branchNameText := "add-app-" + app + "-" + version
	title := fmt.Sprintf("Add %s %s", app, version)
	message := fmt.Sprintf("Add app %s %s", app, version)
	var pullRequestInfo *gits.PullRequestInfo
	if o.FakePullRequests != nil {
		var err error
		pullRequestInfo, err = o.FakePullRequests(o.DevEnv, modifyRequirementsFn, branchNameText, title, message,
			nil)
		if err != nil {
			return err
		}
	} else {
		var err error
		pullRequestInfo, err = o.createEnvironmentPullRequest(o.DevEnv, modifyRequirementsFn, branchNameText, title,
			message,
			nil, o.ConfigureGitCallback)
		if err != nil {
			return err
		}
	}
	log.Infof("Added app via Pull Request %s\n", pullRequestInfo.PullRequest.URL)
	return nil
}

func (o *AddAppOptions) installApp(app string) error {
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that helm is present")
	}
	setValues := strings.Split(o.SetValues, ",")

	err = o.installChart(app, app, o.Version, o.Namespace, o.HelmUpdate, setValues, o.ValueFiles, o.Repo)
	if err != nil {
		return fmt.Errorf("failed to install app %s: %v", app, err)
	}
	return o.exposeApp(app)
}

// TODO Patch this up to use app CRD
func (o *AddAppOptions) exposeApp(addon string) error {
	service, ok := kube.AddonServices[addon]
	if !ok {
		return nil
	}
	svc, err := o.KubeClientCached.CoreV1().Services(o.Namespace).Get(service, meta_v1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "getting the addon service: %s", service)
	}

	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	if svc.Annotations[kube.AnnotationExpose] == "" {
		svc.Annotations[kube.AnnotationExpose] = "true"
		svc, err = o.KubeClientCached.CoreV1().Services(o.Namespace).Update(svc)
		if err != nil {
			return errors.Wrap(err, "updating the service annotations")
		}
	}
	devNamespace, _, err := kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
	if err != nil {
		return errors.Wrap(err, "retrieving the dev namespace")
	}
	return o.expose(devNamespace, o.Namespace, "")
}
