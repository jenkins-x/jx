package cmd

import (
	"github.com/spf13/cobra"
)

// StepPreOptions defines the CLI arguments
type StepPreOptions struct {
	*CommonOptions

	DisableImport bool
	OutDir        string
}

var ()

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepPre(commonOpts *CommonOptions) *cobra.Command {
	options := &StepPreOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:   "pre",
		Short: "pre step actions",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepPreBuild(commonOpts))
	cmd.AddCommand(NewCmdStepPreExtend(commonOpts))

	return cmd
}

// Run implements this command
func (o *StepPreOptions) Run() error {
	return o.Cmd.Help()
}
