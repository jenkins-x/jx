package main

import (
	"io"

	"github.com/spf13/cobra"
)

type configSetCmd struct {
	out   io.Writer
	args  []string
	key   string
	value string
}

func newConfigSetCmd(out io.Writer) *cobra.Command {
	ccmd := &configSetCmd{
		out:  out,
		args: []string{"key", "value"},
	}
	cmd := &cobra.Command{
		Use:   "set",
		Short: "set global Draft configuration stored in $DRAFT_HOME/config.toml",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return ccmd.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return ccmd.run()
		},
	}
	return cmd
}

func (ccmd *configSetCmd) complete(args []string) error {
	if err := validateArgs(args, ccmd.args); err != nil {
		return err
	}
	ccmd.key = args[0]
	ccmd.value = args[1]
	return nil
}

func (ccmd *configSetCmd) run() error {
	globalConfig[ccmd.key] = ccmd.value
	return SaveConfig(globalConfig)
}
