package cmd

import (
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
)

// StepEnvOptions contains the command line flags
type StepEnvOptions struct {
	StepOptions
}

// NewCmdStepEnv Steps a command object for the "step" command
func NewCmdStepEnv(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepEnvOptions{
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
		Use:   "env",
		Short: "env [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepEnvApply(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *StepEnvOptions) Run() error {
	return o.Cmd.Help()
}

