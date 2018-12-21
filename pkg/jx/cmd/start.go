package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// Start contains the command line options
type Start struct {
	CommonOptions
}

var (
	start_long = templates.LongDesc(`
		Starts a process such as a Jenkins pipeline.
`)

	start_example = templates.Examples(`
		# Start a pipeline
		jx start pipeline foo
	`)
)

// NewCmdStart creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdStart(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &Start{
		CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "start TYPE [flags]",
		Short:   "Starts a process such as a pipeline",
		Long:    start_long,
		Example: start_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"begin"},
	}

	cmd.AddCommand(NewCmdStartPipeline(f, in, out, errOut))
	cmd.AddCommand(NewCmdStartProtection(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *Start) Run() error {
	return o.Cmd.Help()
}
