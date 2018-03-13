package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack/repo"
	"github.com/spf13/cobra"
)

type listCmd struct {
	out  io.Writer
	home draftpath.Home
}

func newListCmd(out io.Writer) *cobra.Command {
	list := &listCmd{out: out}

	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "list all installed pack repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			list.home = draftpath.Home(homePath())
			return list.run()
		},
	}
	return cmd
}

func (l *listCmd) run() error {
	repos := repo.FindRepositories(l.home.Packs())
	if len(repos) == 0 {
		return errors.New("no pack repositories to show")
	}
	for i := range repos {
		fmt.Fprintln(l.out, repos[i].Name)
	}
	return nil
}
