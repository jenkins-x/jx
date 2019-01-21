package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// DeleteGitOptions are the flags for delete commands
type DeleteGitOptions struct {
	commoncmd.CommonOptions
}

// NewCmdDeleteGit creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteGit(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteGitOptions{
		commoncmd.CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "git",
		Short: "Deletes one or more Git resources",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm"},
	}

	cmd.AddCommand(NewCmdDeleteGitServer(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteGitToken(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteGitOptions) Run() error {
	return o.Cmd.Help()
}
