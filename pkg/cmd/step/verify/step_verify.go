package verify

import (
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/spf13/cobra"
)

// StepVerifyOptions contains the command line flags
type StepVerifyOptions struct {
	opts.StepOptions
}

func NewCmdStepVerify(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepVerifyOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "verify [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepVerifyGit(commonOpts))
	cmd.AddCommand(NewCmdStepVerifyInstall(commonOpts))
	cmd.AddCommand(NewCmdStepVerifyPod(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepVerifyOptions) Run() error {
	return o.Cmd.Help()
}
