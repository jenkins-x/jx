package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// CreateChatOptions the options for the create spring command
type CreateChatOptions struct {
	CreateOptions
}

// NewCmdCreateChat creates a command object for the "create" command
func NewCmdCreateChat(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateChatOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "chat",
		Short:   "Creates a chat server resource",
		Aliases: []string{"slackr"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateChatServer(f, out, errOut))
	cmd.AddCommand(NewCmdCreateChatToken(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateChatOptions) Run() error {
	return o.Cmd.Help()
}
