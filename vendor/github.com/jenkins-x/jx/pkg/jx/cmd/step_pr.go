package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepPROptions struct {
	StepOptions
}

var ()

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepPR(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepPROptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "pr",
		Short: "pipeline step pr",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepPRComment(f, out, errOut))
	options.addCommonFlags(cmd)

	return cmd
}

// Run implements this command
func (o *StepPROptions) Run() error {
	return o.Cmd.Help()
}
