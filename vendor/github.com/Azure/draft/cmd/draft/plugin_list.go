package main

import (
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

type pluginListCmd struct {
	home draftpath.Home
	out  io.Writer
}

func newPluginListCmd(out io.Writer) *cobra.Command {
	pcmd := &pluginListCmd{out: out}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list installed Draft plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			pcmd.home = draftpath.Home(homePath())
			return pcmd.run()
		},
	}
	return cmd
}

func (pcmd *pluginListCmd) run() error {
	pluginDirs := pluginDirPath(pcmd.home)

	debug("looking for plugins at: %s", pluginDirs)
	plugins, err := findPlugins(pluginDirs)
	if err != nil {
		return err
	}

	if len(plugins) == 0 {
		fmt.Fprintln(pcmd.out, "No plugins found")
		return nil
	}

	table := uitable.New()
	table.AddRow("NAME", "VERSION", "DESCRIPTION")
	for _, p := range plugins {
		table.AddRow(p.Metadata.Name, p.Metadata.Version, p.Metadata.Description)
	}
	fmt.Fprintln(pcmd.out, table)
	return nil
}
