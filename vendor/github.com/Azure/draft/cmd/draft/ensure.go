package main

import (
	"fmt"

	"github.com/Azure/draft/pkg/osutil"

	pluginbase "k8s.io/helm/pkg/plugin"

	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/Azure/draft/pkg/plugin"
)

// ensureDirectories checks to see if $DRAFT_HOME exists
//
// If $DRAFT_HOME does not exist, this function will create it.
func (i *initCmd) ensureDirectories() error {
	configDirectories := []string{
		i.home.String(),
		i.home.Plugins(),
		i.home.Packs(),
		i.home.Logs(),
	}
	for _, p := range configDirectories {
		err := osutil.EnsureDirectory(p)
		if err != nil {
			return err
		}
	}

	return nil
}

// ensureConfig checks to see if $DRAFT_HOME/config.toml exists
//
// If it does not exist, this function will create it.
func (i *initCmd) ensureConfig() error {
	return osutil.EnsureFile(i.home.Config())
}

// ensurePacks checks to see if the default packs exist.
//
// If the pack does not exist, this function will create it.
func (i *initCmd) ensurePacks() error {
	existingRepos := repo.FindRepositories(i.home.Packs())

	fmt.Fprintln(i.out, "Installing default pack repositories...")
	for _, builtin := range repo.Builtins() {
		if err := i.ensurePack(builtin, existingRepos); err != nil {
			return err
		}
	}
	fmt.Fprintln(i.out, "Installation of default pack repositories complete")
	return nil
}

func (i *initCmd) ensurePack(builtin *repo.Builtin, existingRepos []repo.Repository) error {

	for _, repo := range existingRepos {
		if builtin.Name == repo.Name {
			return nil
		}
	}

	addArgs := []string{
		"add",
		builtin.URL,
	}

	addFlags := []string{
		"--version",
		builtin.Version,
		"--home",
		string(i.home),
		fmt.Sprintf("--debug=%v", flagDebug),
	}

	packRepoCmd, _, err := rootCmd.Find([]string{"pack-repo"})
	if err != nil {
		return err
	}

	if err := packRepoCmd.ParseFlags(addFlags); err != nil {
		return err
	}

	if err := packRepoCmd.RunE(packRepoCmd, addArgs); err != nil {
		return err
	}
	debug("Successfully installed pack repo: %v %v", builtin.URL, builtin.Version)
	return nil
}

// ensurePlugins checks to see if the default plugins exist.
//
// If the plugin does not exist, this function will add it.
func (i *initCmd) ensurePlugins() error {

	existingPlugins, err := findPlugins(pluginDirPath(i.home))
	if err != nil {
		return err
	}

	fmt.Fprintln(i.out, "Installing default plugins...")
	for _, builtin := range plugin.Builtins() {
		if err := i.ensurePlugin(builtin, existingPlugins); err != nil {
			return err
		}
	}
	fmt.Fprintln(i.out, "Installation of default plugins complete")
	return nil
}

func (i *initCmd) ensurePlugin(builtin *plugin.Builtin, existingPlugins []*pluginbase.Plugin) error {

	for _, pl := range existingPlugins {
		if builtin.Name == pl.Metadata.Name {
			if builtin.Version == pl.Metadata.Version {
				return nil
			}

			debug("Currently have %v version %v. Removing to install %v",
				pl.Metadata.Name, pl.Metadata.Version, builtin.Version)

			if err := removePlugin(pl); err != nil {
				return err
			}

			debug("Successfully removed %v %v",
				pl.Metadata.Name, pl.Metadata.Version)
		}
	}

	installArgs := []string{
		builtin.URL,
	}

	installFlags := []string{
		"--version",
		builtin.Version,
		"--home",
		string(i.home),
		fmt.Sprintf("--debug=%v", flagDebug),
	}

	plugInstallCmd, _, err := rootCmd.Find([]string{"plugin", "install"})
	if err != nil {
		return err
	}

	if err := plugInstallCmd.ParseFlags(installFlags); err != nil {
		return err
	}

	if err := plugInstallCmd.PreRunE(plugInstallCmd, installArgs); err != nil {
		return err
	}

	if err := plugInstallCmd.RunE(plugInstallCmd, installArgs); err != nil {
		return err
	}

	// reload plugins
	loadPlugins(rootCmd, i.home, i.out, i.in)

	debug("Successfully installed %v %v from %v",
		builtin.Name, builtin.Version, builtin.URL)
	return nil
}
