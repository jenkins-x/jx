package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/version"
)

const versionDesc = `
Show the server version for draftd.

This will print the server version of draftd. The output will look something like
this:

v1.0.0
`

type versionCmd struct {
	out io.Writer
}

func newVersionCmd(out io.Writer) *cobra.Command {
	version := &versionCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "print the server version information",
		Long:  versionDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return version.run()
		},
	}
	return cmd
}

func (v *versionCmd) run() error {
	fmt.Fprintf(v.out, "%s\n", version.New())
	return nil
}
