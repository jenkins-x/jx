package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/spf13/cobra"
)

// StepSyntaxOptions contains the command line flags
type StepSyntaxOptions struct {
	StepOptions
}

// NewCmdStepSyntax Steps a command object for the "step" command
func NewCmdStepSyntax(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepSyntaxOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "syntax",
		Short: "syntax [command]",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepSyntaxValidate(commonOpts))
	cmd.AddCommand(NewCmdStepSyntaxSchema(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepSyntaxOptions) Run() error {
	return o.Cmd.Help()
}
