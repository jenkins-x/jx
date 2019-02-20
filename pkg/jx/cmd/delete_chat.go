package cmd

import (
	"github.com/spf13/cobra"
)

// DeleteChatOptions are the flags for delete commands
type DeleteChatOptions struct {
	*CommonOptions
}

// NewCmdDeleteChat creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteChat(commonOpts *CommonOptions) *cobra.Command {
	options := &DeleteChatOptions{
		commonOpts,
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

	cmd.AddCommand(NewCmdDeleteChatServer(commonOpts))
	cmd.AddCommand(NewCmdDeleteChatToken(commonOpts))
	return cmd
}

// Run implements this command
func (o *DeleteChatOptions) Run() error {
	return o.Cmd.Help()
}
