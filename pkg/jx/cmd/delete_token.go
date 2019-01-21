package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// DeleteTokenOptions are the flags for delete commands
type DeleteTokenOptions struct {
	commoncmd.CommonOptions
}

// NewCmdDeleteToken creates a command object
// retrieves one or more resources from a server.
func NewCmdDeleteToken(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteTokenOptions{
		commoncmd.CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "token",
		Short:   "Deletes one or more issue token resources",
		Aliases: []string{"api-token"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdDeleteTokenAddon(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteTokenOptions) Run() error {
	return o.Cmd.Help()
}
