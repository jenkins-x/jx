package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// DeleteAddonOptions are the flags for delete commands
type DeleteAddonOptions struct {
	CommonOptions
}

// NewCmdDeleteAddon creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteAddon(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteAddonOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "addon",
		Short: "Deletes one or many addons",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm"},
	}

	cmd.AddCommand(NewCmdDeleteAddonGitea(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *DeleteAddonOptions) Run() error {
	return o.Cmd.Help()
}
