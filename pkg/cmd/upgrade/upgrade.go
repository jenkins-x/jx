package upgrade

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Upgrades all of the plugins in your local Jenkins X CLI
`)

	cmdExample = templates.Examples(`
		# upgrades your plugin binaries
		jx upgrade
	`)
)

// UpgradeOptions the options for upgrading a cluster
type Options struct {
	Cmd *cobra.Command
}

// NewCmdUpgrade creates a command object for the command
func NewCmdUpgrade() (*cobra.Command, *Options) {
	o := &Options{}

	o.Cmd = &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrades resources",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	o.Cmd.AddCommand(cobras.SplitCommand(NewCmdUpgradeCLI()))
	o.Cmd.AddCommand(cobras.SplitCommand(NewCmdUpgradePlugins()))

	return o.Cmd, o
}

// Run implements this command
func (o *Options) Run() error {
	return o.Cmd.Help()
}
