package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepPreOptions defines the CLI arguments
type StepPreOptions struct {
	commoncmd.CommonOptions

	DisableImport bool
	OutDir        string
}

var ()

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepPre(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepPreOptions{
		CommonOptions: commoncmd.CommonOptions{
			Factory: f,
			In:      in,
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

	cmd.AddCommand(NewCmdStepPreBuild(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepPreExtend(f, in, out, errOut))

	return cmd
}

// Run implements this command
func (o *StepPreOptions) Run() error {
	return o.Cmd.Help()
}
