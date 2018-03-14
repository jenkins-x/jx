package main

import (
	"fmt"
	"io"

	"github.com/Azure/draft/pkg/draft"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

const logsDesc = `This command outputs logs from the draft server to help debug builds.`

type logsCmd struct {
	client   *draft.Client
	out      io.Writer
	appName  string
	buildID  string
	logLines int64
	args     []string
}

func newLogsCmd(out io.Writer) *cobra.Command {
	lc := &logsCmd{
		out:  out,
		args: []string{"app-name", "build-id"},
	}

	cmd := &cobra.Command{
		Use:   "logs <app-name> <build-id>",
		Short: logsDesc,
		Long:  logsDesc,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := setupConnection(cmd, args); err != nil {
				return err
			}
			return lc.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			lc.client = ensureDraftClient(lc.client)
			return lc.run()
		},
	}

	f := cmd.Flags()
	f.Int64Var(&lc.logLines, "tail", 100, "lines of recent log lines to display")

	return cmd
}

func (l *logsCmd) complete(args []string) error {
	if err := validateArgs(args, l.args); err != nil {
		return err
	}
	l.appName = args[0]
	l.buildID = args[1]
	return nil
}

func (l *logsCmd) run() error {
	b, err := l.client.GetLogs(context.Background(), l.appName, l.buildID, draft.WithLogsLimit(l.logLines))
	if err != nil {
		return err
	}
	fmt.Fprint(l.out, string(b))
	return nil
}
