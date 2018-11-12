package cmd

import (
	"github.com/spf13/cobra"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepReportOptions struct {
	StepOptions
}

var ()

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepReport(commonOpts *CommonOptions) *cobra.Command {
	options := &StepReportOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
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

	cmd.AddCommand(NewCmdStepReportActivities(commonOpts))
	cmd.AddCommand(NewCmdStepReportReleases(commonOpts))

	return cmd
}

// Run implements this command
func (o *StepReportOptions) Run() error {
	return o.Cmd.Help()
}
