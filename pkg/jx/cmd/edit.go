package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
)

// EditOptions contains the CLI options
type EditOptions struct {
	*CommonOptions
}

var (
	exit_long = templates.LongDesc(`
		Edit a resource

`)

	exit_example = templates.Examples(`
		# Lets edit the staging Environment
		jx edit env staging
	`)
)

// NewCmdEdit creates the edit command
func NewCmdEdit(commonOpts *CommonOptions) *cobra.Command {
	options := &EditOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "edit [flags]",
		Short:   "Edit a resource",
		Long:    exit_long,
		Example: exit_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"modify"},
	}

	cmd.AddCommand(NewCmdCreateBranchPattern(commonOpts))
	cmd.AddCommand(NewCmdEditAddon(commonOpts))
	cmd.AddCommand(NewCmdEditBuildpack(commonOpts))
	cmd.AddCommand(NewCmdEditConfig(commonOpts))
	cmd.AddCommand(NewCmdEditEnv(commonOpts))
	cmd.AddCommand(NewCmdEditHelmBin(commonOpts))
	cmd.AddCommand(NewCmdEditStorage(commonOpts))
	cmd.AddCommand(NewCmdEditUserRole(commonOpts))
	cmd.AddCommand(NewCmdEditExtensionsRepository(commonOpts))
	addTeamSettingsCommandsFromTags(cmd, commonOpts.In, commonOpts.Out, commonOpts.Err, options)
	return cmd
}

// Run implements this command
func (o *EditOptions) Run() error {
	return o.Cmd.Help()
}
