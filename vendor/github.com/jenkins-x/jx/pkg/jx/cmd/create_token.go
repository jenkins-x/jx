package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// CreateTokenOptions the options for the create spring command
type CreateTokenOptions struct {
	CreateOptions
}

// NewCmdCreateToken creates a command object for the "create" command
func NewCmdCreateToken(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateTokenOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
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
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateTokenAddon(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateTokenOptions) Run() error {
	return o.Cmd.Help()
}
