package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/local"
	"github.com/hpcloud/tail"
	"github.com/spf13/cobra"
)

const logsDesc = `This command outputs logs from the draft server to help debug builds.`

var (
	runningEnvironment string
)

type logsCmd struct {
	out     io.Writer
	appName string
	buildID string
	line    uint
	tail    bool
	args    []string
	home    draftpath.Home
}

func newLogsCmd(out io.Writer) *cobra.Command {
	lc := &logsCmd{
		out:  out,
		args: []string{"build-id"},
	}

	cmd := &cobra.Command{
		Use:     "logs <build-id>",
		Short:   logsDesc,
		Long:    logsDesc,
		PreRunE: lc.complete,
		RunE: func(cmd *cobra.Command, args []string) error {
			deployedApp, err := local.DeployedApplication(draftToml, runningEnvironment)
			if err != nil {
				return err
			}
			lc.appName = deployedApp.Name
			b, err := getLatestBuildID(lc.appName)
			if err != nil {
				return fmt.Errorf("cannot get latest build: %v", err)
			}
			lc.buildID = b

			if len(args) > 0 {
				lc.buildID = args[0]
			}
			return lc.run(cmd, args)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&lc.tail, "tail", false, "tail the logs file as it's being written")
	f.UintVar(&lc.line, "line", 20, "line location to tail from (offset from end of file)")
	f.StringVarP(&runningEnvironment, environmentFlagName, environmentFlagShorthand, defaultDraftEnvironment(), environmentFlagUsage)
	return cmd
}

func (l *logsCmd) complete(_ *cobra.Command, args []string) error {
	l.home = draftpath.Home(homePath())
	return nil
}

func (l *logsCmd) run(_ *cobra.Command, _ []string) error {
	if l.tail {
		return l.tailLogs(int64(l.line))
	}
	return l.dumpLogs()
}

func (l *logsCmd) dumpLogs() error {
	f, err := os.Open(filepath.Join(l.home.Logs(), l.appName, l.buildID))
	if err != nil {
		return fmt.Errorf("could not read logs for %s: %v", l.buildID, err)
	}
	defer f.Close()
	io.Copy(l.out, f)
	return nil
}

func (l *logsCmd) tailLogs(offset int64) error {
	t, err := tail.TailFile(filepath.Join(l.home.Logs(), l.appName, l.buildID), tail.Config{
		Location: &tail.SeekInfo{Offset: -offset, Whence: os.SEEK_END},
		Logger:   tail.DiscardingLogger,
		Follow:   true,
		ReOpen:   true,
	})
	if err != nil {
		return err
	}
	for line := range t.Lines {
		fmt.Fprintln(l.out, line.Text)
	}
	return t.Wait()
}
