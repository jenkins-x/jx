package create

import (
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/spf13/cobra"
)

// StepCreateOptions contains the command line flags
type StepCreateOptions struct {
	opts.StepOptions
}

// NewCmdStepCreate Steps a command object for the "step" command
func NewCmdStepCreate(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreateOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "create [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepCreateDevPodWorkpace(commonOpts))
	cmd.AddCommand(NewCmdStepCreateJenkinsConfig(commonOpts))
	cmd.AddCommand(NewCmdStepCreateTask(commonOpts))
	cmd.AddCommand(NewCmdStepCreateInstallValues(commonOpts))
	cmd.AddCommand(NewCmdStepCreateVersionPullRequest(commonOpts))
	cmd.AddCommand(NewCmdStepCreateValues(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepCreateOptions) Run() error {
	return o.Cmd.Help()
}
