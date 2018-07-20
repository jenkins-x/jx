package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

// DeleteChatOptions are the flags for delete commands
type DeleteChatOptions struct {
	CommonOptions
}

// NewCmdDeleteChat creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteChat(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteChatOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "chat",
		Short:   "Deletes one or many chat services resources",
		Aliases: []string{"slack"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdDeleteChatServer(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteChatToken(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteChatOptions) Run() error {
	return o.Cmd.Help()
}
