package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// CreateAddonOptions the options for the create spring command
type CreateAddonOptions struct {
	CreateOptions
}

// NewCmdCreateAddon creates a command object for the "create" command
func NewCmdCreateAddon(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "addon",
		Short:   "Creates an addon",
		Aliases: []string{"scm"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateAddonGitea(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateAddonOptions) Run() error {
	return o.Cmd.Help()
}
