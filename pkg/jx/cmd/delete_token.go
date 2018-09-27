package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// DeleteTokenOptions are the flags for delete commands
type DeleteTokenOptions struct {
	CommonOptions
}

// NewCmdDeleteToken creates a command object
// retrieves one or more resources from a server.
func NewCmdDeleteToken(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteTokenOptions{
		CommonOptions{
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
