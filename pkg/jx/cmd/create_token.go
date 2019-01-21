package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateTokenOptions the options for the create spring command
type CreateTokenOptions struct {
	CreateOptions
}

// NewCmdCreateToken creates a command object for the "create" command
func NewCmdCreateToken(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateTokenOptions{
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
		Use:     "token",
		Short:   "Creates a new user token for a service",
		Aliases: []string{"api-token", "password", "pwd"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateTokenAddon(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateTokenOptions) Run() error {
	return o.Cmd.Help()
}
