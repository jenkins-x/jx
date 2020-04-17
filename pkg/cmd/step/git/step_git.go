package git

import (
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/v2/pkg/cmd/step/git/credentials"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
)

// StepGitOptions contains the command line flags
type StepGitOptions struct {
	step.StepOptions
}

// NewCmdStepGit Steps a command object for the "step" command
func NewCmdStepGit(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepGitOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "git",
		Short: "git [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.AddCommand(credentials.NewCmdStepGitCredentials(commonOpts))
	cmd.AddCommand(NewCmdStepGitEnvs(commonOpts))
	cmd.AddCommand(NewCmdStepGitMerge(commonOpts))
	cmd.AddCommand(NewCmdStepGitForkAndClone(commonOpts))
	cmd.AddCommand(NewCmdStepGitValidate(commonOpts))
	cmd.AddCommand(NewCmdStepGitClose(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepGitOptions) Run() error {
	return o.Cmd.Help()
}
