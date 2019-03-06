package cmd

import (
	"github.com/spf13/cobra"
)

// StepVerifyPodOptions contains the command line flags
type StepVerifyPodOptions struct {
	StepOptions
}

// NewCmdStepVerifyPod creates the `jx step verify pod` command
func NewCmdStepVerifyPod(commonOpts *CommonOptions) *cobra.Command {

	options := &StepVerifyPodOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "pod",
		Short: "pod [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepVerifyPodCount(commonOpts))
	cmd.AddCommand(NewCmdStepVerifyPodReady(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepVerifyPodOptions) Run() error {
	return o.Cmd.Help()
}
