package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/local"
)

const logsDesc = `This command outputs logs from the draft server to help debug builds.`

type logsCmd struct {
	out      io.Writer
	logLines int64
}

func newLogsCmd(out io.Writer) *cobra.Command {
	lc := &logsCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: logsDesc,
		Long:  logsDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return lc.run()
		},
	}

	f := cmd.Flags()
	f.Int64Var(&lc.logLines, "tail", 100, "lines of recent log lines to display")

	return cmd
}

func (l *logsCmd) run() error {
	client, clientConfig, err := getKubeClient(kubeContext)
	if err != nil {
		return fmt.Errorf("Could not get a kube client: %s", err)
	}
	restClientConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("Could not retrieve client config from the kube client: %s", err)
	}

	draftApp := &local.App{
		Name:      "draftd",
		Namespace: "kube-system",
		Container: "draftd",
	}

	connection, err := draftApp.Connect(client, restClientConfig)
	if err != nil {
		return fmt.Errorf("Could not connect to draftd: %s", err)
	}

	fmt.Fprintf(l.out, "Starting a log stream from the draft server...\n")
	readCloser, err := connection.RequestLogStream(draftApp, l.logLines)
	if err != nil {
		return fmt.Errorf("Could not get log stream: %s", err)
	}

	defer readCloser.Close()
	_, err = io.Copy(l.out, readCloser)
	if err != nil {
		return err
	}

	return nil
}
