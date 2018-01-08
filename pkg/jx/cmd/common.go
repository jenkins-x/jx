package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/spf13/cobra"
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
