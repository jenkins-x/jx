package main

import (
	"errors"
	"fmt"
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/Azure/draft/pkg/draft"
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
	out        io.Writer
	client     *draft.Client
	short      bool
	clientOnly bool
	serverOnly bool
}

func newVersionCmd(out io.Writer) *cobra.Command {
	version := &versionCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "print the client version information",
		Long:  versionDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !version.clientOnly {
				// We do this manually instead of in PreRun because we only
				// need a tunnel if server version is requested.
				setupConnection(cmd, args)
			}
			version.client = ensureDraftClient(version.client)
			return version.run()
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&version.clientOnly, "client", "c", false, "client version only")
	f.BoolVarP(&version.serverOnly, "server", "s", false, "server version only")

	return cmd
}

func (v *versionCmd) run() error {
	if !v.serverOnly {
		cv := version.New()
		fmt.Fprintf(v.out, "Client: %s\n", formatVersion(cv, v.short))
	}

	if v.clientOnly {
		return nil
	}

	sv, err := v.client.Version(context.Background())
	if err != nil {
		log.Debug(err)
		return errors.New("cannot connect to draftd")
	}
	fmt.Fprintf(v.out, "Server: %s\n", formatVersion(sv, v.short))
	return nil
}

func formatVersion(v *version.Version, short bool) string {
	if short {
		return fmt.Sprintf("%s+g%s", v.SemVer, v.GitCommit[:7])
	}
	return fmt.Sprintf("%#v", v)
}
