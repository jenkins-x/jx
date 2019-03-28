package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/apps"

	"github.com/jenkins-x/jx/pkg/environments"

	"github.com/jenkins-x/jx/pkg/io/secrets"

	"github.com/jenkins-x/jx/pkg/util"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback environments.ConfigureGitFn

	Namespace   string
	Version     string
	ReleaseName string
	SetValues   []string
	ValuesFiles []string
	HelmUpdate  bool
}

const (
	optionHelmUpdate = "helm-update"
	optionValues     = "values"
	optionSet        = "set"
	optionAlias      = "alias"
)

// NewCmdAddApp creates a command object for the "create" command
func NewCmdAddApp(commonOpts *CommonOptions) *cobra.Command {
	options := &AddAppOptions{
		AddOptions: AddOptions{
			CommonOptions: commonOpts,
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

	options.addFlags(cmd, kube.DefaultNamespace)
	return cmd
}

func (o *AddAppOptions) addFlags(cmd *cobra.Command, defaultNamespace string) {

	// Common flags

	cmd.Flags().StringVarP(&o.Version, "version", "v", "",
		"The chart version to install")
	cmd.Flags().StringVarP(&o.Repo, "repository", "", "",
		"The repository from which the app should be installed (default specified in your dev environment)")
	cmd.Flags().StringVarP(&o.Username, "username", "", "",
		"The username for the repository")
	cmd.Flags().StringVarP(&o.Password, "password", "", "",
		"The password for the repository")
	cmd.Flags().StringVarP(&o.Alias, optionAlias, "", "",
		"An alias to use for the app if you wish to install multiple instances of the same app")
	cmd.Flags().StringVarP(&o.ReleaseName, optionRelease, "r", "",
		"The chart release name (by default the name of the app, available when NOT using GitOps for your dev"+
			" environment)")
	cmd.Flags().BoolVarP(&o.HelmUpdate, optionHelmUpdate, "", true,
		"Should we run helm update first to ensure we use the latest version (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringVarP(&o.Namespace, optionNamespace, "n", "", "The Namespace to install into (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringArrayVarP(&o.ValuesFiles, optionValues, "f", []string{}, "List of locations for values files, "+
		"can be local files or URLs (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringArrayVarP(&o.SetValues, optionSet, "s", []string{},
		"The chart set values (can specify multiple or separate values with commas: key1=val1,key2=val2) (available when NOT using GitOps for your dev environment)")

}

// Run implements this command
func (o *AddAppOptions) Run() error {
	o.GitOps, o.DevEnv = o.GetDevEnv()
	if o.Repo == "" {
		o.Repo = o.DevEnv.Spec.TeamSettings.AppsRepository
	}
	if o.Repo == "" {
		o.Repo = kube.DefaultChartMuseumURL
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
		msg := "unable to specify --%s when using GitOps for your dev environment"
		if o.ReleaseName != "" {
			return util.InvalidOptionf(optionRelease, o.ReleaseName, msg, optionRelease)
		}
		if !o.HelmUpdate {
			return util.InvalidOptionf(optionHelmUpdate, o.HelmUpdate, msg, optionHelmUpdate)
		}
		if o.Namespace != "" && o.Namespace != kube.DefaultNamespace {
			return util.InvalidOptionf(optionNamespace, o.Namespace, msg, optionNamespace)
		}
		if len(o.SetValues) > 0 {
			return util.InvalidOptionf(optionSet, o.SetValues, msg, optionSet)
		}
		if len(o.ValuesFiles) > 1 {
			return util.InvalidOptionf(optionValues, o.SetValues,
				"no more than one --%s can be specified when using GitOps for your dev environment", optionValues)
		}
		if o.GetSecretsLocation() != secrets.VaultLocationKind {
			return fmt.Errorf("cannot install apps without a vault when using GitOps for your dev environment")
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
		err := o.ensureHelm()
		if err != nil {
			return errors.Wrap(err, "failed to ensure that helm is present")
		}
		jxClient, ns, err := o.JXClientAndDevNamespace()
		if err != nil {
			return errors.Wrapf(err, "getting jx client")
		}
		kubeClient, _, err := o.KubeClientAndDevNamespace()
		if err != nil {
			return errors.Wrapf(err, "getting kubeClient")
		}
		if o.Namespace == "" {
			o.Namespace = ns
		}

		if o.Alias != "" && o.ReleaseName == "" {
			bin, noTiller, helmTemplate, err := o.TeamHelmBin()
			if err != nil {
				return err
			}
			if bin != "helm" || noTiller || helmTemplate {
				o.ReleaseName = o.Alias
			} else {
				o.ReleaseName = o.Alias + "-" + o.Namespace
			}
		}
		opts.Namespace = o.Namespace
		opts.KubeClient = kubeClient
		opts.JxClient = jxClient
		opts.InstallTimeout = defaultInstallTimeout
	}
	if o.GetSecretsLocation() == secrets.VaultLocationKind {
		teamName, _, err := o.TeamAndEnvironmentNames()
		if err != nil {
			return err
		}
		opts.TeamName = teamName
		client, err := o.SystemVaultClient("")
		if err != nil {
			return err
		}
		opts.VaultClient = &client
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

	var version string
	if o.Version != "" {
		version = o.Version
	}
	app := args[0]
	if o.ReleaseName == "" {
		o.ReleaseName = app
	}

	return opts.AddApp(app, version, o.Repo, o.Username, o.Password, o.ReleaseName, o.ValuesFiles, o.SetValues,
		o.Alias, o.HelmUpdate)
}
