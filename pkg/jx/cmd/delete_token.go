package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// DeleteTokenOptions are the flags for delete commands
type DeleteTokenOptions struct {
	CommonOptions
}

// NewCmdDeleteToken creates a command object
// retrieves one or more resources from a server.
func NewCmdDeleteToken(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteTokenOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "token",
		Short:   "Deletes one or many issue token resources",
		Aliases: []string{"api-token"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdDeleteTokenAddon(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteTokenOptions) Run() error {
	return o.Cmd.Help()
}
