package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/util"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// AddAppOptions the options for the create spring command
type AddAppOptions struct {
	AddOptions

	GitOps bool
	DevEnv *jenkinsv1.Environment

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

	o.GitOps, o.DevEnv = o.GetDevEnv()

	// Common flags

	cmd.Flags().StringVarP(&o.Version, "version", "v", defaultVersion,
		"The chart version to install")
	cmd.Flags().StringVarP(&o.Repo, "repository", "", o.DevEnv.Spec.TeamSettings.AppsRepository,
		"The repository from which the app should be installed")
	cmd.Flags().StringVarP(&o.Username, "username", "", "",
		"The username for the repository")
	cmd.Flags().StringVarP(&o.Password, "password", "", "",
		"The password for the repository")
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
		cmd.Flags().StringVarP(&o.SetValues, "set", "s", "",
			"The chart set values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	}

}

// Run implements this command
func (o *AddAppOptions) Run() error {
	// Regiser the App CRD
	apiClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterAppCRD(apiClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the App CRD")
	}

	args := o.Args
	if len(args) == 0 {
		return o.Cmd.Help()
	}
	if len(args) > 1 {
		return o.Cmd.Help()
	}

	if o.Repo == "" {
		return fmt.Errorf("must specify a repository")
	}

	for _, arg := range args {
		version := o.Version
		if version == "" {
			var err error
			version, err = helm.GetLatestVersion(arg, o.Repo, o.Username, o.Password, o.Helm())
			if err != nil {
				return err
			}
			if o.Verbose {
				log.Infof("No version specified so using latest version which is %s\n", util.ColorInfo(version))
			}
		}
		if o.GitOps {
			err := o.createPR(arg, version)
			if err != nil {
				return err
			}
		} else {
			err := o.installApp(arg, version)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func (o *AddAppOptions) createPR(app string, version string) error {

	modifyRequirementsFn := func(requirements *helm.Requirements) error {
		// See if the app already exists in requirements
		found := false
		for _, d := range requirements.Dependencies {
			if d.Name == app && d.Alias == o.Alias {
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
		pullRequestInfo, err = o.createEnvironmentPullRequest(o.DevEnv, modifyRequirementsFn, &branchNameText, &title,
			&message,
			nil, o.ConfigureGitCallback)
		if err != nil {
			return err
		}
	}
	log.Infof("Added app via Pull Request %s\n", pullRequestInfo.PullRequest.URL)
	return nil
}

func (o *AddAppOptions) installApp(app string, version string) error {
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that helm is present")
	}
	setValues := strings.Split(o.SetValues, ",")

	chart := helm.InstallChartOptions{
		ReleaseName: app,
		Chart:       app,
		Version:     version,
		Ns:          o.Namespace,
		HelmUpdate:  o.HelmUpdate,
		SetValues:   setValues,
		ValueFiles:  o.ValueFiles,
		Repository:  o.Repo,
		Username:    o.Username,
		Password:    o.Password,
	}

	err = o.installChartOptions(chart)
	if err != nil {
		return fmt.Errorf("failed to install app %s: %v", app, err)
	}
	return o.OnAppInstall(app, version)
}
