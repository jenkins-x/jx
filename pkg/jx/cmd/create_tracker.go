package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// CreateTrackerOptions the options for the create spring command
type CreateTrackerOptions struct {
	CreateOptions
}

// NewCmdCreateTracker creates a command object for the "create" command
func NewCmdCreateTracker(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateTrackerOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "tracker",
		Short:   "Creates an issue tracker resource",
		Aliases: []string{"jra", "trello", "issue-tracker"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateTrackerServer(f, out, errOut))
	cmd.AddCommand(NewCmdCreateTrackerToken(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateTrackerOptions) Run() error {
	return o.Cmd.Help()
}
