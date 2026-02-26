package create

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

// InstallDependenciesFlags flags for the install command
type InstallDependenciesFlags struct {
	Dependencies []string
	All          bool
}

// InstallDependenciesOptions options for install dependencies
type InstallDependenciesOptions struct {
	*opts.CommonOptions
	Flags InstallDependenciesFlags
}

var (
	installDependenciesLong = templates.LongDesc(`
		Installs required dependencies
`)

	installDependenciesExample = templates.Examples(`
		# Install dependencies
		jx install dependencies

		jx install dependencies -d gcloud
`)

	availableDependencies = []string{
		"az",
		"kubectl",
		"gcloud",
		"helm",
		"tiller",
		"helm3",
		"ksync",
		"oc",
		"aws",
		"eksctl",
		"aws-iam-authenticator",
		"kustomize",
	}
)

// NewCmdInstallDependencies creates a command object to install any required dependencies
func NewCmdInstallDependencies(commonOpts *opts.CommonOptions) *cobra.Command {

	options := CreateInstallDependenciesOptions(commonOpts)

	cmd := &cobra.Command{
		Use:     "dependencies",
		Short:   "Install missing dependencies",
		Long:    installDependenciesLong,
		Example: installDependenciesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
		SuggestFor: []string{"dependency"},
	}

	cmd.Flags().StringArrayVarP(&options.Flags.Dependencies, "dependencies", "d", []string{}, "The dependencies to install")
	cmd.Flags().BoolVarP(&options.Flags.All, "all", "", false, "Install all dependencies")

	return cmd
}

// CreateInstallDependenciesOptions creates the options for jx install dependencies
func CreateInstallDependenciesOptions(commonOpts *opts.CommonOptions) InstallDependenciesOptions {
	options := InstallDependenciesOptions{
		CommonOptions: commonOpts,
	}
	return options
}

// Run implements this command
func (options *InstallDependenciesOptions) Run() error {
	install := []string{}

	if !options.Flags.All {
		if len(options.Flags.Dependencies) == 0 {

			prompt := &survey.MultiSelect{
				Message: "What dependencies would you like to install:",
				Options: availableDependencies,
			}
			surveyOpts := survey.WithStdio(options.In, options.Out, options.Err)
			err := survey.AskOne(prompt, &install, nil, surveyOpts)
			if err != nil {
				return err
			}
		} else {
			install = append(install, options.Flags.Dependencies...)
		}
	} else {
		install = availableDependencies
		options.NoBrew = true
	}

	if len(install) > 0 {
		return options.DoInstallMissingDependencies(install)
	}

	log.Logger().Debugf("No dependencies selected to install")
	return nil
}
