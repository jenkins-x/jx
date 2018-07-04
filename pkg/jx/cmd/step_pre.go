package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

// StepPreOptions defines the CLI arguments
type StepPreOptions struct {
	CommonOptions

	DisableImport bool
	OutDir        string
}

var ()

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepPre(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepPreOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
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

	cmd.AddCommand(NewCmdStepPreBuild(f, out, errOut))

	return cmd
}

// Run implements this command
func (o *StepPreOptions) Run() error {
	return o.Cmd.Help()
}
