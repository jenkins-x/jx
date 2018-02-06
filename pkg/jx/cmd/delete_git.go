package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// DeleteGitOptions are the flags for delete commands
type DeleteGitOptions struct {
	CommonOptions
}

// NewCmdDeleteGit creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteGit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteGitOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "git",
		Short: "Deletes one or many git resources",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm"},
	}

	cmd.AddCommand(NewCmdDeleteGitServer(f, out, errOut))
	cmd.AddCommand(NewCmdDeleteGitToken(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteGitOptions) Run() error {
	return o.Cmd.Help()
}
