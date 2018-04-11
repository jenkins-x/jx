package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// DeleteChatOptions are the flags for delete commands
type DeleteChatOptions struct {
	CommonOptions
}

// NewCmdDeleteChat creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteChat(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
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
			cmdutil.CheckErr(err)
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
