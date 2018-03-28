package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/draftpath"
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

	return cmd
}

// runInit initializes local config and installs Draft to Kubernetes Cluster
func (i *initCmd) run() error {

	if !i.dryRun {
		if err := i.setupDraftHome(); err != nil {
			return err
		}
	}

	fmt.Fprintf(i.out, "$DRAFT_HOME has been configured at %s.\nHappy Sailing!\n", draftHome)
	return nil
}

func (i *initCmd) setupDraftHome() error {
	ensureFuncs := []func() error{
		i.ensureDirectories,
		i.ensureConfig,
		i.ensurePlugins,
		i.ensurePacks,
	}

	for _, funct := range ensureFuncs {
		if err := funct(); err != nil {
			return err
		}
	}

	return nil
}
