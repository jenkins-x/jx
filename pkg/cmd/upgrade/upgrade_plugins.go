package upgrade

import (
	"fmt"
	"os"
	"regexp"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx/pkg/plugins"

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
		"gitops":    true,
		"health":    true,
		"secret":    true,
		"verify":    true,
		"changelog": true,
	}
)

// UpgradeOptions the options for upgrading a cluster
type PluginOptions struct {
	CommandRunner cmdrunner.CommandRunner
	OnlyMandatory bool
	Boot          bool
	Path          string
}

// NewCmdUpgrade creates a command object for the command
func NewCmdUpgradePlugins() (*cobra.Command, *PluginOptions) {
	o := &PluginOptions{}

	cmd := &cobra.Command{
		Use:     "plugins",
		Short:   "Upgrades all of the plugins in your local Jenkins X CLI",
		Long:    cmdPluginsLong,
		Example: cmdPluginsExample,
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&o.OnlyMandatory, "mandatory", "m", false, "if set lets ignore optional plugins")
	cmd.Flags().BoolVarP(&o.Boot, "boot", "", false, "only install plugins required for boot")
	cmd.Flags().StringVarP(&o.Path, "path", "", "/usr/bin", "creates a symlink to the binary plugins in this bin path dir")

	return cmd, o
}

// Run implements the command
func (o *PluginOptions) Run() error {
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return fmt.Errorf("failed to find plugin bin directory: %w", err)
	}

	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	for k := range plugins.Plugins {
		p := plugins.Plugins[k]
		if o.Boot && !bootPlugins[p.Name] {
			continue
		}
		log.Logger().Infof("checking binary jx plugin %s version %s is installed", termcolor.ColorInfo(p.Name), termcolor.ColorInfo(p.Spec.Version))
		fileName, err := extensions.EnsurePluginInstalled(p, pluginBinDir)
		if err != nil {
			return fmt.Errorf("failed to ensure plugin is installed %s: %w", p.Name, err)
		}

		if o.Boot {
			if p.Name == "gitops" {
				c := &cmdrunner.Command{
					Name: fileName,
					Args: []string{"plugin", "upgrade", "--path", o.Path},
				}
				_, err = o.CommandRunner(c)
				if err != nil {
					return fmt.Errorf("failed to upgrade gitops plugin %s: %w", p.Name, err)
				}
			}
			continue
		}

		// TODO we could use metadata on the plugin for this?
		if p.Name == "secret" {
			c := &cmdrunner.Command{
				Name: fileName,
				Args: []string{"plugins", "upgrade"},
			}
			_, err = o.CommandRunner(c)
			if err != nil {
				return fmt.Errorf("failed to upgrade plugin %s: %w", p.Name, err)
			}
		}
	}
	if !(o.OnlyMandatory || o.Boot) {
		// Upgrade the rest
		file, err := os.Open(pluginBinDir)
		if err != nil {
			return fmt.Errorf("failed to read plugin dir %s: %w", pluginBinDir, err)
		}
		defer file.Close()
		files, err := file.Readdirnames(0)
		if err != nil {
			return fmt.Errorf("failed to read plugin dir %s: %w", pluginBinDir, err)
		}
		pluginPattern := regexp.MustCompile("^(jx-.*)-[0-9.]+$")
		extraPlugins := make(map[string]bool)
		for _, plugin := range files {
			res := pluginPattern.FindStringSubmatch(plugin)
			if len(res) > 1 {
				cleanPlugin := res[1]
				if plugins.PluginMap[cleanPlugin] == nil {
					extraPlugins[cleanPlugin] = true
				}
			}
		}
		for plugin := range extraPlugins {
			_, err = plugins.InstallStandardPlugin(pluginBinDir, plugin)
			if err != nil {
				log.Logger().Warnf("Failed to upgrade plugin %s: %+v", plugin, err)
			}
		}
	}

	return nil
}
