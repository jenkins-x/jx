package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// DeleteTrackerOptions are the flags for delete commands
type DeleteTrackerOptions struct {
	CommonOptions
}

// NewCmdDeleteTracker creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteTracker(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteTrackerOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "tracker",
		Short:   "Deletes one or many issue tracker resources",
		Aliases: []string{"jra", "trello", "issue-tracker"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdDeleteTrackerServer(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteTrackerToken(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteTrackerOptions) Run() error {
	return o.Cmd.Help()
}
