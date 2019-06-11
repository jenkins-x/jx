package create

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/spf13/cobra"
)

// CreateChatOptions the options for the create spring command
type CreateChatOptions struct {
	CreateOptions
}

// NewCmdCreateChat creates a command object for the "create" command
func NewCmdCreateChat(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateChatOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
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
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateChatServer(commonOpts))
	cmd.AddCommand(NewCmdCreateChatToken(commonOpts))
	return cmd
}

// Run implements this command
func (o *CreateChatOptions) Run() error {
	return o.Cmd.Help()
}
