package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepPROptions struct {
	StepOptions
}

var ()

// NewCmdStepPR Steps a command object for the "step pr" command
func NewCmdStepPR(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepPROptions{
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
		Use:   "pr",
		Short: "pipeline step pr",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepPRComment(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepPRLabels(f, in, out, errOut))
	options.addCommonFlags(cmd)

	return cmd
}

// Run implements this command
func (o *StepPROptions) Run() error {
	return o.Cmd.Help()
}
