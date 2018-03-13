package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/Azure/draft/pkg/draft/pack/repo/installer"
	"github.com/spf13/cobra"
)

type updateCmd struct {
	out    io.Writer
	err    io.Writer
	source string
	home   draftpath.Home
}

func newUpdateCmd(out io.Writer) *cobra.Command {
	upd := &updateCmd{out: out}

	cmd := &cobra.Command{
		Use:   "update [flags]",
		Short: "fetch the newest version of all pack repositories using git.",
		RunE: func(cmd *cobra.Command, args []string) error {
			upd.home = draftpath.Home(homePath())
			return upd.run()
		},
	}
	return cmd
}

func (upd *updateCmd) run() error {

	repos := repo.FindRepositories(upd.home.Packs())
	if len(repos) == 0 {
		fmt.Fprintf(upd.out, "No pack repositories found to update. All up to date!")
		return nil
	}
	var updatedRepoNames []string
	for i := range repos {
		exactLocation, err := filepath.EvalSymlinks(repos[i].Dir)
		if err != nil {
			return err
		}
		absExactLocation, err := filepath.Abs(exactLocation)
		if err != nil {
			return err
		}

		ins, err := installer.FindSource(absExactLocation, upd.home)
		if err != nil {
			return err
		}
		if err := installer.Update(ins); err != nil {
			return err
		}
		updatedRepoNames = append(updatedRepoNames, repos[i].Name)
	}
	repoMsg := "updated %d pack repository %v\n"
	if len(repos) > 1 {
		repoMsg = "updated %d pack repositories %v\n"
	}
	fmt.Fprintf(upd.out, repoMsg, len(repos), updatedRepoNames)
	return nil
}
