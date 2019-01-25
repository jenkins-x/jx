package cmd

import (
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
)

// StepCreateOptions contains the command line flags
type StepCreateOptions struct {
	StepOptions
}

// NewCmdStepCreate Steps a command object for the "step" command
func NewCmdStepCreate(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepCreateOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "create [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepCreateBuild(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepCreateBuildTemplate(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepCreateTask(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *StepCreateOptions) Run() error {
	return o.Cmd.Help()
}
