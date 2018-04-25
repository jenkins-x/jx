package main

import (
	"fmt"
	"io"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/pack"
	"github.com/spf13/cobra"
)

type packListCmd struct {
	out  io.Writer
	repo string         // list packs from this repo
	home draftpath.Home // $DRAFT_HOME
}

func newPackListCmd(out io.Writer) *cobra.Command {
	list := &packListCmd{out: out}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list available Draft packs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list.run()
		},
	}
	list.home = draftpath.Home(homePath())
	cmd.Flags().StringVar(&list.repo, "repo", "", "list packs by repo (default all)")
	return cmd
}

func (cmd *packListCmd) run() error {
	packs, err := pack.List(cmd.home.Packs(), cmd.repo)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.out, "Available Packs:")
	for _, name := range packs {
		fmt.Fprintf(cmd.out, "  %s\n", name)
	}
	return nil
}
