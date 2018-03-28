package main

import (
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

type configListCmd struct {
	out io.Writer
}

func newConfigListCmd(out io.Writer) *cobra.Command {
	ccmd := &configListCmd{out: out}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list global Draft configuration stored in $DRAFT_HOME/config.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ccmd.run()
		},
	}
	return cmd
}

func (ccmd *configListCmd) run() error {
	table := uitable.New()
	table.AddRow("KEY", "VALUE")
	for k, v := range globalConfig {
		table.AddRow(k, v)
	}
	fmt.Fprintln(ccmd.out, table)
	return nil
}
