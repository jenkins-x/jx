package main

import (
	"io"

	"github.com/spf13/cobra"
)

type configUnsetCmd struct {
	out  io.Writer
	args []string
	key  string
}

func newConfigUnsetCmd(out io.Writer) *cobra.Command {
	ccmd := &configUnsetCmd{
		out:  out,
		args: []string{"key"},
	}
	cmd := &cobra.Command{
		Use:   "unset",
		Short: "unset global Draft configuration stored in $DRAFT_HOME/config.toml",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return ccmd.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return ccmd.run()
		},
	}
	return cmd
}

func (ccmd *configUnsetCmd) complete(args []string) error {
	if err := validateArgs(args, ccmd.args); err != nil {
		return err
	}
	ccmd.key = args[0]
	return nil
}

func (ccmd *configUnsetCmd) run() error {
	delete(globalConfig, ccmd.key)
	return SaveConfig(globalConfig)
}
