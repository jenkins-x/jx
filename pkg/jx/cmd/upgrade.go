package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// UpgradeOptions are the flags for delete commands
type UpgradeOptions struct {
	CommonOptions
}

var (
	upgrade_long = templates.LongDesc(`
		Upgrade the whole Jenkins X platform.
`)

	upgrade_example = templates.Examples(`
		# upgrade the command line tools 
		jx upgrade cli

		# upgrade the platform 
		jx upgrade platform

		# upgrade extensions
		jx upgrade extensions 
	`)
)

// NewCmdUpgrade creates the command
func NewCmdUpgrade(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpgradeOptions{
		CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "upgrade [flags]",
		Short:   "Upgrades a resource",
		Long:    upgrade_long,
		Example: upgrade_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"update"},
	}

	cmd.AddCommand(NewCmdUpgradeAddons(f, in, out, errOut))
	cmd.AddCommand(NewCmdUpgradeCLI(f, in, out, errOut))
	cmd.AddCommand(NewCmdUpgradeBinaries(f, in, out, errOut))
	cmd.AddCommand(NewCmdUpgradeCluster(f, in, out, errOut))
	cmd.AddCommand(NewCmdUpgradeIngress(f, in, out, errOut))
	cmd.AddCommand(NewCmdUpgradePlatform(f, in, out, errOut))
	cmd.AddCommand(NewCmdUpgradeExtensions(f, in, out, errOut))
	/* TODO fails TestNewJXCommand
	cmd.AddCommand(NewCmdUpgradeApps(f, in, out, errOut))
	*/
	return cmd
}

// Run implements this command
func (o *UpgradeOptions) Run() error {
	return o.Cmd.Help()
}
