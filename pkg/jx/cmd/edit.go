package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// EditOptions contains the CLI options
type EditOptions struct {
	CommonOptions
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
func NewCmdEdit(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &EditOptions{
		CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
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

	cmd.AddCommand(NewCmdCreateBranchPattern(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditAddon(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditBuildpack(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditConfig(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditEnv(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditHelmBin(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditStorage(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditUserRole(f, in, out, errOut))
	cmd.AddCommand(NewCmdEditExtensionsRepository(f, in, out, errOut))
	addTeamSettingsCommandsFromTags(cmd, in, out, errOut, options)
	return cmd
}

// Run implements this command
func (o *EditOptions) Run() error {
	return o.Cmd.Help()
}
