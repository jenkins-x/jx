package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/helm/pkg/plugin"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin/installer"
)

type pluginInstallCmd struct {
	source  string
	version string
	home    draftpath.Home
	out     io.Writer
	args    []string
}

func newPluginInstallCmd(out io.Writer) *cobra.Command {
	pcmd := &pluginInstallCmd{
		out:  out,
		args: []string{"plugin"},
	}

	cmd := &cobra.Command{
		Use:   "install [options] <path|url>...",
		Short: "install one or more Draft plugins",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return pcmd.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return pcmd.run()
		},
	}
	cmd.Flags().StringVar(&pcmd.version, "version", "", "specify a version constraint. If this is not specified, the latest version is installed")
	return cmd
}

func (pcmd *pluginInstallCmd) complete(args []string) error {
	if err := validateArgs(args, pcmd.args); err != nil {
		return err
	}
	pcmd.source = args[0]
	pcmd.home = draftpath.Home(homePath())
	return nil
}

func (pcmd *pluginInstallCmd) run() error {
	installer.Debug = flagDebug

	i, err := installer.New(pcmd.source, pcmd.version, pcmd.home)
	if err != nil {
		return err
	}

	debug("installing plugin from %s", pcmd.source)
	if err := installer.Install(i); err != nil {
		return err
	}

	debug("loading plugin from %s", i.Path())
	p, err := plugin.LoadDir(i.Path())
	if err != nil {
		return err
	}

	debug("running any install instructions for plugin: %s", p.Metadata.Name)
	if err := runHook(p, plugin.Install); err != nil {
		return err
	}

	fmt.Fprintf(pcmd.out, "Installed plugin: %s\n", p.Metadata.Name)
	return nil
}
