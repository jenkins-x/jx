package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
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
func NewCmdStart(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &Start{
		CommonOptions{
			Factory: f,
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
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	cmd.AddCommand(NewCmdStartPipeline(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *Start) Run() error {
	return o.Cmd.Help()
}
