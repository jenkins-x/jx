package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

type configGetCmd struct {
	out  io.Writer
	args []string
	key  string
}

func newConfigGetCmd(out io.Writer) *cobra.Command {
	ccmd := &configGetCmd{
		out:  out,
		args: []string{"key"},
	}
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get global Draft configuration stored in $DRAFT_HOME/config.toml",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return ccmd.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return ccmd.run()
		},
	}
	return cmd
}

func (ccmd *configGetCmd) complete(args []string) error {
	if err := validateArgs(args, ccmd.args); err != nil {
		return err
	}
	ccmd.key = args[0]
	return nil
}

func (ccmd *configGetCmd) run() error {
	v, ok := globalConfig[ccmd.key]
	if !ok {
		return errors.New("specified key could not be found in config")
	}

	fmt.Fprintln(ccmd.out, v)
	return nil
}
