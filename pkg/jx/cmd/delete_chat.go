package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// DeleteChatOptions are the flags for delete commands
type DeleteChatOptions struct {
	commoncmd.CommonOptions
}

// NewCmdDeleteChat creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteChat(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteChatOptions{
		commoncmd.CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "chat",
		Short:   "Deletes one or more chat services resources",
		Aliases: []string{"slack"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdDeleteChatServer(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteChatToken(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteChatOptions) Run() error {
	return o.Cmd.Help()
}
