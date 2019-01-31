package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// ControllerOptions contains the CLI options
type ControllerOptions struct {
	CommonOptions
}

var (
	controllerLong = templates.LongDesc(`
		Runs a controller

`)

	controllerExample = templates.Examples(`
	`)
)

// NewCmdController creates the edit command
func NewCmdController(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &ControllerOptions{
		CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "controller <command> [flags]",
		Short:   "Runs a controller",
		Long:    controllerLong,
		Example: controllerExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdControllerBackup(f, in, out, errOut))
	cmd.AddCommand(NewCmdControllerBuild(f, in, out, errOut))
	cmd.AddCommand(NewCmdControllerBuildNumbers(f, in, out, errOut))
	cmd.AddCommand(NewCmdControllerRole(f, in, out, errOut))
	cmd.AddCommand(NewCmdControllerTeam(f, in, out, errOut))
	cmd.AddCommand(NewCmdControllerWorkflow(f, in, out, errOut))
	cmd.AddCommand(NewCmdControllerCommitStatus(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *ControllerOptions) Run() error {
	return o.Cmd.Help()
}
