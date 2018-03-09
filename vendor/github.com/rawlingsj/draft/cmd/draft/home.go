package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const homeDesc = `
Display the location of DRAFT_HOME. This is where any Draft configuration files live.
`

func newHomeCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "home",
		Short: "print the location of DRAFT_HOME",
		Long:  homeDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(out, homePath())
			return nil
		},
	}
	return cmd
}
