package upgrade

import (
	"github.com/jenkins-x/jx-cli/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdPluginsLong = templates.LongDesc(`
		Upgrades all of the plugins in your local Jenkins X CLI
`)

	cmdPluginsExample = templates.Examples(`
		# upgrades your plugin binaries
		jx upgrade plugins
	`)

	bootPlugins = map[string]bool{
		"gitops": true,
		"secret": true,
		"verify": true,
	}
)

// UpgradeOptions the options for upgrading a cluster
type PluginOptions struct {
	CommandRunner cmdrunner.CommandRunner
	OnlyMandatory bool
	Boot          bool
}

// NewCmdUpgrade creates a command object for the command
func NewCmdUpgradePlugins() (*cobra.Command, *PluginOptions) {
	o := &PluginOptions{}

	cmd := &cobra.Command{
		Use:     "plugins",
		Short:   "Upgrades all of the plugins in your local Jenkins X CLI",
		Long:    cmdPluginsLong,
		Example: cmdPluginsExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&o.OnlyMandatory, "mandatory", "m", false, "if set lets ignore optional plugins")
	cmd.Flags().BoolVarP(&o.Boot, "boot", "", false, "only install plugins required for boot")

	return cmd, o
}

// Run implements the command
func (o *PluginOptions) Run() error {
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return errors.Wrap(err, "failed to find plugin bin directory")
	}

	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	for k := range plugins.Plugins {
		p := plugins.Plugins[k]
		if o.Boot && !bootPlugins[p.Name] {
			continue
		}
		if o.OnlyMandatory && p.Name == "jenkins" {
			continue
		}
		log.Logger().Infof("checking binary jx plugin %s version %s is installed", termcolor.ColorInfo(p.Name), termcolor.ColorInfo(p.Spec.Version))
		fileName, err := extensions.EnsurePluginInstalled(p, pluginBinDir)
		if err != nil {
			return errors.Wrapf(err, "failed to ensure plugin is installed %s", p.Name)
		}

		// TODO we could use metadata on the plugin for this?
		switch p.Name {
		case "admin", "secret":
			if p.Name == "secret" {
				c := &cmdrunner.Command{
					Name: fileName,
					Args: []string{"plugins", "upgrade"},
				}
				_, err = o.CommandRunner(c)
				if err != nil {
					return errors.Wrapf(err, "failed to upgrade plugin %s", p.Name)
				}
			}
		}
	}
	return nil
}
