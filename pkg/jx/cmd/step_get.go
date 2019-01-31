package cmd

import (
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
)

// StepGetOptions contains the command line flags
type StepGetOptions struct {
	StepOptions
}

// NewCmdStepGet Steps a command object for the "step" command
func NewCmdStepGet(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepGetOptions{
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
		Use:   "get",
		Short: "get [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepGetBuildNumber(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *StepGetOptions) Run() error {
	return o.Cmd.Help()
}
