package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/helm/pkg/plugin"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin/installer"
)

type pluginUpdateCmd struct {
	names []string
	home  draftpath.Home
	out   io.Writer
}

func newPluginUpdateCmd(out io.Writer) *cobra.Command {
	pcmd := &pluginUpdateCmd{out: out}
	cmd := &cobra.Command{
		Use:   "update <plugin>...",
		Short: "update one or more Draft plugins",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return pcmd.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return pcmd.run()
		},
	}
	return cmd
}

func (pcmd *pluginUpdateCmd) complete(args []string) error {
	if len(args) == 0 {
		return errors.New("please provide plugin name to update")
	}
	pcmd.names = args
	pcmd.home = draftpath.Home(homePath())
	return nil
}

func (pcmd *pluginUpdateCmd) run() error {
	installer.Debug = flagDebug
	pluginsDir := pluginDirPath(pcmd.home)
	debug("loading installed plugins from %s", pluginsDir)
	plugins, err := findPlugins(pluginsDir)
	if err != nil {
		return err
	}
	var errorPlugins []string

	for _, name := range pcmd.names {
		if found := findPlugin(plugins, name); found != nil {
			if err := updatePlugin(found, pcmd.home); err != nil {
				errorPlugins = append(errorPlugins, fmt.Sprintf("Failed to update plugin %s, got error (%v)", name, err))
			} else {
				fmt.Fprintf(pcmd.out, "Updated plugin: %s\n", name)
			}
		} else {
			errorPlugins = append(errorPlugins, fmt.Sprintf("Plugin: %s not found", name))
		}
	}
	if len(errorPlugins) > 0 {
		return fmt.Errorf(strings.Join(errorPlugins, "\n"))
	}
	return nil
}

func updatePlugin(p *plugin.Plugin, home draftpath.Home) error {
	exactLocation, err := filepath.EvalSymlinks(p.Dir)
	if err != nil {
		return err
	}
	absExactLocation, err := filepath.Abs(exactLocation)
	if err != nil {
		return err
	}

	i, err := installer.FindSource(absExactLocation, home)
	if err != nil {
		return err
	}
	if err := installer.Update(i); err != nil {
		return err
	}

	debug("loading plugin from %s", i.Path())
	updatedPlugin, err := plugin.LoadDir(i.Path())
	if err != nil {
		return err
	}

	return runHook(updatedPlugin, plugin.Update)
}
