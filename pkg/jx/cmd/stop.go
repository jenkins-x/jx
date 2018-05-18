package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// Stop contains the command line options
type Stop struct {
	CommonOptions
}

var (
	stopLong = templates.LongDesc(`
		Stops a process such as a Jenkins pipeline.
`)

	stopExample = templates.Examples(`
		# Stop a pipeline
		jx stop pipeline foo
	`)
)

// NewCmdStop creates the command object
func NewCmdStop(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &Stop{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "stop TYPE [flags]",
		Short:   "Stops a process such as a pipeline",
		Long:    stopLong,
		Example: stopExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	cmd.AddCommand(NewCmdStopPipeline(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *Stop) Run() error {
	return o.Cmd.Help()
}
