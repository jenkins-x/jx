package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepReportOptions struct {
	StepOptions
}

var ()

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepReport(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepReportOptions{
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
		Use:   "report",
		Short: "report step actions",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepReportActivities(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepReportReleases(f, in, out, errOut))

	return cmd
}

// Run implements this command
func (o *StepReportOptions) Run() error {
	return o.Cmd.Help()
}
