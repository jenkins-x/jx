package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Azure/draft-pack-repo/version"
)

const versionDesc = `
Displays version information.
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
		Short: "display version information",
		Long:  versionDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return version.run()
		},
	}
	return cmd
}

func (v *versionCmd) run() error {
	ver := version.New()
	fmt.Fprintln(v.out, ver.String())
	return nil
}
