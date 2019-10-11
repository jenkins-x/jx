package github

import (
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/spf13/cobra"
)

// StepVerifyOptions contains the command line flags
type StepGithubOptions struct {
	step.StepOptions
}

func NewCmdStepGithub(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepGithubOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "github",
		Short: "github [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepGithubApp(commonOpts))

	return cmd
}

// Run implements this command
func (o *StepGithubOptions) Run() error {
	return o.Cmd.Help()
}
