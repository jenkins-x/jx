package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateChatOptions the options for the create spring command
type CreateChatOptions struct {
	CreateOptions
}

// NewCmdCreateChat creates a command object for the "create" command
func NewCmdCreateChat(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateChatOptions{
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
		Use:     "chat",
		Short:   "Creates a chat server resource",
		Aliases: []string{"slackr"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateChatServer(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateChatToken(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateChatOptions) Run() error {
	return o.Cmd.Help()
}
