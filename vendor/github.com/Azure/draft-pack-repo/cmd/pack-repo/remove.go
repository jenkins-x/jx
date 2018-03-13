package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/spf13/cobra"
)

type removeCmd struct {
	in       io.Reader
	out      io.Writer
	repoName string
	home     draftpath.Home
}

func newRemoveCmd(out io.Writer, in io.Reader) *cobra.Command {
	rm := &removeCmd{out: out, in: in}

	cmd := &cobra.Command{
		Use:   "remove [flags] <name>",
		Short: "remove a pack repository",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return rm.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := promptYesOrNo("Are you sure you want to do this?", false, rm.in, rm.out)
			if err != nil {
				return err
			}
			if ok {
				return rm.run()
			}
			return nil
		},
	}
	return cmd
}

func (rm *removeCmd) complete(args []string) error {
	if err := validateArgs(args, []string{"name"}); err != nil {
		return err
	}
	rm.repoName = args[0]
	rm.home = draftpath.Home(homePath())
	return nil
}

func (rm *removeCmd) run() error {
	r := repo.Repository{
		Name: rm.repoName,
		Dir:  filepath.Join(rm.home.Packs(), rm.repoName),
	}

	var found = false
	repos := repo.FindRepositories(rm.home.Packs())
	for i := range repos {
		if repos[i].Name == r.Name {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("pack repository %s not found", r.Name)
	}

	if err := os.RemoveAll(r.Dir); err != nil {
		return err
	}

	fmt.Fprintf(rm.out, "removed pack repository %s\n", r.Name)
	return nil
}
