package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepPROptions struct {
	StepOptions
}

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepPR(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepPROptions{
		StepOptions: StepOptions{
			CommonOptions: commoncmd.CommonOptions{
				Factory: f,
				In:      in,
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
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepPRComment(f, in, out, errOut))
	options.AddCommonFlags(cmd)

	return cmd
}

// Run implements this command
func (o *StepPROptions) Run() error {
	return o.Cmd.Help()
}
