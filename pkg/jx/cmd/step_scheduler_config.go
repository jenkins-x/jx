package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
)

// StepSchedulerConfigOptions contains the command line flags
type StepSchedulerConfigOptions struct {
	StepOptions
}

var (
	stepSchedulerConfigLong = templates.LongDesc(`
		This pipeline step command allows you to work with the scheduler configuration. Sub commands include:

		* jx step scheduler config apply
`)
)

// NewCmdStepSchedulerConfig Steps a command object for the "step" command
func NewCmdStepSchedulerConfig(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepSchedulerConfigOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "config",
		Short: "scheduler config [command]",
		Long:  stepSchedulerConfigLong,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepSchedulerConfigApply(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepSchedulerConfigOptions) Run() error {
	return o.Cmd.Help()
}
