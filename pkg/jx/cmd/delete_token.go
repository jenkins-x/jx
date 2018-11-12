package cmd

import (
	"github.com/spf13/cobra"
)

// DeleteTokenOptions are the flags for delete commands
type DeleteTokenOptions struct {
	*CommonOptions
}

// NewCmdDeleteToken creates a command object
// retrieves one or more resources from a server.
func NewCmdDeleteToken(commonOpts *CommonOptions) *cobra.Command {
	options := &DeleteTokenOptions{
		CommonOptions: commonOpts,
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

	cmd.AddCommand(NewCmdDeleteTokenAddon(commonOpts))
	return cmd
}

// Run implements this command
func (o *DeleteTokenOptions) Run() error {
	return o.Cmd.Help()
}
