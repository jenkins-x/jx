package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateTrackerOptions the options for the create spring command
type CreateTrackerOptions struct {
	CreateOptions
}

// NewCmdCreateTracker creates a command object for the "create" command
func NewCmdCreateTracker(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateTrackerOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commoncmd.CommonOptions{
				Factory: f,
				In:      in,
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
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateTrackerServer(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateTrackerToken(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateTrackerOptions) Run() error {
	return o.Cmd.Help()
}
