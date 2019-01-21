package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// DeleteTrackerOptions are the flags for delete commands
type DeleteTrackerOptions struct {
	commoncmd.CommonOptions
}

// NewCmdDeleteTracker creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteTracker(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteTrackerOptions{
		commoncmd.CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "tracker",
		Short:   "Deletes one or more issue tracker resources",
		Aliases: []string{"jra", "trello", "issue-tracker"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdDeleteTrackerServer(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteTrackerToken(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteTrackerOptions) Run() error {
	return o.Cmd.Help()
}
