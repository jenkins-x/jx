package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// UpgradeOptions are the flags for delete commands
type UpgradeOptions struct {
	CommonOptions
}

// NewCmdUpgrade creates the command
func NewCmdUpgrade(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &UpgradeOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "upgrade [flags]",
		Short:   "Upgrades a resource",
		Long:    delete_long,
		Example: delete_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm"},
	}

	cmd.AddCommand(NewCmdUpgradePlatform(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *UpgradeOptions) Run() error {
	return o.Cmd.Help()
}
