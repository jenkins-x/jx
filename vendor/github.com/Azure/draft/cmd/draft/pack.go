package main

import (
	"io"

	"github.com/spf13/cobra"
)

const (
	packHelp = `Manage Draft packs stored in $DRAFT_HOME/packs.`
)

func newPackCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "manage Draft packs",
		Long:  packHelp,
	}
	cmd.AddCommand(
		newPackListCmd(out),
	)
	return cmd
}
