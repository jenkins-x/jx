package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateGitOptions the options for the create spring command
type CreateGitOptions struct {
	CreateOptions
}

// NewCmdCreateGit creates a command object for the "create" command
func NewCmdCreateGit(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateGitOptions{
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
		Use:     "git",
		Short:   "Creates a Git resource",
		Aliases: []string{"scm"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateGitServer(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateGitToken(f, in, out, errOut))
	cmd.AddCommand(NewCmdCreateGitUser(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateGitOptions) Run() error {
	return o.Cmd.Help()
}
