package github

import (
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/spf13/cobra"
)

// StepVerifyOptions contains the command line flags
type StepGithubAppOptions struct {
	step.StepOptions
}

func NewCmdStepGithubApp(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepGithubOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "app",
		Short: "github app [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepGithubAppToken(commonOpts))

	return cmd
}

// Run implements this command
func (o *StepGithubAppOptions) Run() error {
	return o.Cmd.Help()
}
