package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// CreateGitOptions the options for the create spring command
type CreateGitOptions struct {
	CreateOptions
}

// NewCmdCreateGit creates a command object for the "create" command
func NewCmdCreateGit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateGitOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "git",
		Short:   "Creates a git resource",
		Aliases: []string{"scm"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateGitServer(f, out, errOut))
	cmd.AddCommand(NewCmdCreateGitToken(f, out, errOut))
	cmd.AddCommand(NewCmdCreateGitUser(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateGitOptions) Run() error {
	return o.Cmd.Help()
}
