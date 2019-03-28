package cmd

import (
	"github.com/jenkins-x/jx/pkg/apps"

	"github.com/jenkins-x/jx/pkg/environments"

	"github.com/pkg/errors"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
)

var (
	deleteAppLong = templates.LongDesc(`
		Deletes one or more Apps

`)

	deleteAppExample = templates.Examples(`
		# prompt for the available apps to delete
		jx delete apps 

		# delete a specific app 
		jx delete app jx-app-cheese
	`)
)

const (
	optionPurge = "purge"
)

// DeleteAppOptions are the flags for this delete commands
type DeleteAppOptions struct {
	*CommonOptions

	GitOps bool
	DevEnv *jenkinsv1.Environment

	ReleaseName string
	Namespace   string
	Purge       bool
	Alias       string

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback environments.ConfigureGitFn
}

// NewCmdDeleteApp creates a command object for this command
func NewCmdDeleteApp(commonOpts *CommonOptions) *cobra.Command {
	o := &DeleteAppOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "app",
		Short:   "Deletes one or more apps from Jenkins X",
		Long:    deleteAppLong,
		Example: deleteAppExample,
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&o.ReleaseName, optionRelease, "r", "",
		"The chart release name (available when NOT using GitOps for your dev environment)")
	cmd.Flags().BoolVarP(&o.Purge, optionPurge, "", true,
		"Should we run helm update first to ensure we use the latest version (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringVarP(&o.Namespace, optionNamespace, "n", defaultNamespace, "The Namespace to install into (available when NOT using GitOps for your dev environment)")
	cmd.Flags().StringVarP(&o.Alias, optionAlias, "", "",
		"An alias to use for the app (available when using GitOps for your dev environment)")

	return cmd
}

// Run implements this command
func (o *DeleteAppOptions) Run() error {
	o.GitOps, o.DevEnv = o.GetDevEnv()

	opts := apps.InstallOptions{
		In:        o.In,
		DevEnv:    o.DevEnv,
		Verbose:   o.Verbose,
		Err:       o.Err,
		Out:       o.Out,
		GitOps:    o.GitOps,
		BatchMode: o.BatchMode,
		Helmer:    o.Helm(),
	}

	if o.GitOps {
		msg := "Unable to specify --%s when using GitOps for your dev environment"
		if o.ReleaseName != "" {
			return util.InvalidOptionf(optionRelease, o.ReleaseName, msg, optionRelease)
		}
		if o.Namespace != "" {
			return util.InvalidOptionf(optionNamespace, o.Namespace, msg, optionNamespace)
		}
		gitProvider, _, err := o.createGitProviderForURLWithoutKind(o.DevEnv.Spec.Source.URL)
		if err != nil {
			return errors.Wrapf(err, "creating git provider for %s", o.DevEnv.Spec.Source.URL)
		}
		environmentsDir, err := o.EnvironmentsDir()
		if err != nil {
			return errors.Wrapf(err, "getting environments dir")
		}
		opts.GitProvider = gitProvider
		opts.Gitter = o.Git()
		opts.EnvironmentsDir = environmentsDir
		opts.ConfigureGitFn = o.ConfigureGitCallback
	}
	if !o.GitOps {
		err := o.ensureHelm()
		if err != nil {
			return errors.Wrap(err, "failed to ensure that helm is present")
		}

		if o.Alias != "" {
			return util.InvalidOptionf(optionAlias, o.Alias,
				"Unable to specify --%s when NOT using GitOps for your dev environment", optionAlias)
		}
	}

	args := o.Args
	if len(args) == 0 {
		return o.Cmd.Help()
	}
	if len(args) > 1 {
		return o.Cmd.Help()
	}

	app := args[0]

	return opts.DeleteApp(app, o.Alias, o.ReleaseName, o.Purge)
}
