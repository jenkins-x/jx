package report

import (
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/spf13/cobra"
)

// StepReportOptions contains the command line flags and other helper objects
type StepReportOptions struct {
	step.StepOptions
	OutputDir string
}

// NewCmdStepReport Creates a new Command object
func NewCmdStepReport(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepReportOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "report",
		Short: "report [kind]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepReportChart(commonOpts))
	cmd.AddCommand(NewCmdStepReportJUnit(commonOpts))
	return cmd
}

// AddReportFlags adds common report flags
func (o *StepReportOptions) AddReportFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.OutputDir, "out-dir", "o", "", "The directory to store the resulting reports in")
}

// Run implements this command
func (o *StepReportOptions) Run() error {
	return o.Cmd.Help()
}
