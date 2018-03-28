package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/version"
)

const versionDesc = `
Show the version for draft.

This prints the client and server versions of draft. The output will look something like
this:

Client: &version.Version{SemVer:"v0.1.0", GitCommit:"4f97233d2cc2c7017b07f94211e55bb2670f990d", GitTreeState:"clean"}
Server: &version.Version{SemVer:"v0.1.0", GitCommit:"4f97233d2cc2c7017b07f94211e55bb2670f990d", GitTreeState:"clean"}
`

type versionCmd struct {
	out   io.Writer
	short bool
}

func newVersionCmd(out io.Writer) *cobra.Command {
	versionCmd := &versionCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "print the client version information",
		Long:  versionDesc,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(versionCmd.out, formatVersion(version.New(), versionCmd.short))
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&versionCmd.short, "short", "s", false, "shorten output version")

	return cmd
}

func formatVersion(v *version.Version, short bool) string {
	if short {
		return fmt.Sprintf("%s+g%s", v.SemVer, v.GitCommit[:7])
	}
	return fmt.Sprintf("%#v", v)
}
