package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/io/secrets"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/environments"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	upgradeAppsLong = templates.LongDesc(`
		Upgrades Apps to newer releases
`)

	upgradeAppsExample = templates.Examples(`
		# Upgrade all apps 
		jx upgrade apps
 
        # Upgrade a specific app
        jx upgrade app cheese
	`)
)

// UpgradeAppsOptions the options for the create spring command
type UpgradeAppsOptions struct {
	AddOptions

	GitOps bool
	DevEnv *jenkinsv1.Environment

	Repo        string
	Alias       string
	Username    string
	Password    string
	ReleaseName string

	HelmUpdate bool
	AskAll     bool

	Version string
	All     bool

	Namespace string
	Set       []string

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback environments.ConfigureGitFn

	InstallFlags InstallFlags
}

// NewCmdUpgradeApps defines the command
func NewCmdUpgradeApps(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	o := &UpgradeAppsOptions{
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
		Use:     "apps",
		Short:   "Upgrades any Apps to the latest release",
		Aliases: []string{"app"},
		Long:    upgradeAppsLong,
		Example: upgradeAppsExample,
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&o.BatchMode, optionBatchMode, "b", false, "In batch mode the command never prompts for user input")
	cmd.Flags().BoolVarP(&o.Verbose, optionVerbose, "", false, "Enable verbose logging")
	cmd.Flags().StringVarP(&o.Version, "username", "", "",
		"The username for the repository")
	cmd.Flags().StringVarP(&o.Version, "password", "", "",
		"The password for the repository")
	cmd.Flags().StringVarP(&o.Repo, "repository", "", "",
		"The repository from which the app should be installed")
	cmd.Flags().StringVarP(&o.Alias, "alias", "", "", "An alias to use for the app [--gitops]")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "",
		"The chart version to install [--gitops]")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "The Namespace to promote to [--no-gitops]")
	cmd.Flags().StringArrayVarP(&o.Set, "set", "s", []string{},
		"The Helm parameters to pass in while upgrading [--no-gitops]")
	cmd.Flags().BoolVarP(&o.HelmUpdate, optionHelmUpdate, "", true,
		"Should we run helm update first to ensure we use the latest version (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringVarP(&o.ReleaseName, optionRelease, "r", "",
		"The chart release name (by default the name of the app, available when NOT using GitOps for your dev environment)")
	cmd.Flags().BoolVarP(&o.AskAll, "ask-all", "", false, "Ask all configuration questions. "+
		"By default existing answers are reused automatically.")
	return cmd
}

// Run implements the command
func (o *UpgradeAppsOptions) Run() error {
	o.GitOps, o.DevEnv = o.GetDevEnv()
	if o.Repo == "" {
		o.Repo = o.DevEnv.Spec.TeamSettings.AppsRepository
	}

	opts := apps.InstallOptions{
		In:        o.In,
		DevEnv:    o.DevEnv,
		Verbose:   o.Verbose,
		Err:       o.Err,
		Out:       o.Out,
		GitOps:    o.GitOps,
		BatchMode: o.BatchMode,

		Helmer: o.Helm(),
	}

	if o.GitOps {
		msg := "Unable to specify --%s when using GitOps for your dev environment"
		if o.Namespace != "" {
			return util.InvalidOptionf(optionNamespace, o.ReleaseName, msg, optionNamespace)
		}
		if len(o.Set) > 0 {
			return util.InvalidOptionf(optionSet, o.ReleaseName, msg, optionSet)
		}
		if o.SecretsLocation() != secrets.VaultLocationKind {
			return fmt.Errorf("cannot install apps without a vault when using GitOps for your dev environment")
		}
		if !o.HelmUpdate {
			return util.InvalidOptionf(optionHelmUpdate, o.HelmUpdate, msg, optionHelmUpdate)
		}
		environmentsDir, err := o.EnvironmentsDir()
		if err != nil {
			return errors.Wrapf(err, "getting environments dir")
		}
		opts.EnvironmentsDir = environmentsDir

		gitProvider, _, err := o.createGitProviderForURLWithoutKind(o.DevEnv.Spec.Source.URL)
		if err != nil {
			return errors.Wrapf(err, "creating git provider for %s", o.DevEnv.Spec.Source.URL)
		}
		opts.GitProvider = gitProvider
		opts.ConfigureGitFn = o.ConfigureGitCallback
		opts.Gitter = o.Git()
	}
	if !o.GitOps {
		msg := "Unable to specify --%s when NOT using GitOps for your dev environment"
		if o.Alias != "" {
			return util.InvalidOptionf(optionAlias, o.ReleaseName, msg, optionAlias)
		}
		if o.Version != "" {
			return util.InvalidOptionf(optionVersion, o.ReleaseName, msg, optionVersion)
		}
		jxClient, _, err := o.JXClientAndDevNamespace()
		if err != nil {
			return errors.Wrapf(err, "getting jx client")
		}
		kubeClient, _, err := o.KubeClientAndDevNamespace()
		if err != nil {
			return errors.Wrapf(err, "getting kubeClient")
		}
		opts.Namespace = o.Namespace
		opts.KubeClient = kubeClient
		opts.JxClient = jxClient
		opts.InstallTimeout = defaultInstallTimeout
	}

	if o.SecretsLocation() == secrets.VaultLocationKind {
		teamName, _, err := o.TeamAndEnvironmentNames()
		if err != nil {
			return err
		}
		opts.TeamName = teamName
		client, err := o.CreateSystemVaultClient("")
		if err != nil {
			return err
		}
		opts.VaultClient = &client
	}

	app := ""
	if len(o.Args) > 1 {
		return o.Cmd.Help()
	} else if len(o.Args) == 1 {
		app = o.Args[0]
	}

	var version string
	if o.Version != "" {
		version = o.Version
	}

	return opts.UpgradeApp(app, version, o.Repo, o.Username, o.Password, o.ReleaseName, o.Alias, o.HelmUpdate, o.AskAll)

}
