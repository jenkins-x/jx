package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"fmt"
)

// CommonOptions contains common options and helper methods
type CommonOptions struct {
	Factory cmdutil.Factory
	Out     io.Writer
	Err     io.Writer
	Cmd     *cobra.Command
	Args    []string
}

func (c *CommonOptions) CreateTable() table.Table {
	return c.Factory.CreateTable(c.Out)
}

func (c *CommonOptions) Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(c.Out, format, a...)
}
