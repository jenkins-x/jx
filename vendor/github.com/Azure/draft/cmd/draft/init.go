package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/Azure/draft/pkg/plugin"
)

const (
	initDesc = `
This command sets up local configuration in $DRAFT_HOME (default ~/.draft/) with default set of packs, plugins, and other directories required to work with Draft
`
)

type initCmd struct {
	clientOnly bool
	dryRun     bool
	out        io.Writer
	in         io.Reader
	home       draftpath.Home
	configFile string
}

func newInitCmd(out io.Writer, in io.Reader) *cobra.Command {
	i := &initCmd{
		out: out,
		in:  in,
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "sets up local environment to work with Draft",
		Long:  initDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("This command does not accept arguments")
			}
			i.home = draftpath.Home(homePath())
			return i.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&i.dryRun, "dry-run", false, "go through all the steps without actually installing anything. Mostly used along with --debug for debugging purposes.")
	f.StringVarP(&i.configFile, "config", "f", "", "specify default plugins and pack repositories in a TOML file")

	return cmd
}

// runInit initializes local config and installs Draft to Kubernetes Cluster
func (i *initCmd) run() error {

	pluginOverrides := []plugin.Builtin{}
	repoOverrides := []repo.Builtin{}
	if i.configFile != "" {
		var err error
		pluginOverrides, repoOverrides, err = parseConfigFile(i.configFile)
		if err != nil {
			return fmt.Errorf("Could not parse config file: %s", err)
		}
	}

	if !i.dryRun {
		if err := i.setupDraftHome(pluginOverrides, repoOverrides); err != nil {
			return err
		}
	}

	fmt.Fprintf(i.out, "$DRAFT_HOME has been configured at %s.\nHappy Sailing!\n", draftHome)
	return nil
}

func (i *initCmd) setupDraftHome(plugins []plugin.Builtin, repos []repo.Builtin) error {
	ensureFuncs := []func() error{
		i.ensureDirectories,
		i.ensureConfig,
	}

	for _, funct := range ensureFuncs {
		if err := funct(); err != nil {
			return err
		}
	}

	if err := i.ensurePlugins(plugins); err != nil {
		return err
	}
	if err := i.ensurePacks(repos); err != nil {
		return err
	}

	return nil
}

type obj struct {
	Name    string `toml:"name"`
	URL     string `toml:"url"`
	Version string `toml:"version"`
}

func parseConfigFile(f string) ([]plugin.Builtin, []repo.Builtin, error) {
	var conf map[string][]obj

	if _, err := toml.DecodeFile(f, &conf); err != nil {
		return nil, nil, err
	}

	plugins := []plugin.Builtin{}
	for _, pl := range conf["plugin"] {
		p := plugin.Builtin{
			Name:    pl.Name,
			Version: pl.Version,
			URL:     pl.URL,
		}
		plugins = append(plugins, p)

	}

	repos := []repo.Builtin{}
	for _, re := range conf["repo"] {
		r := repo.Builtin{
			Name:    re.Name,
			Version: re.Version,
			URL:     re.URL,
		}
		repos = append(repos, r)
	}

	return plugins, repos, nil
}
