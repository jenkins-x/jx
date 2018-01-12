package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"os"
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

func (o *CommonOptions) runCommand(name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	err := e.Run()
	if err != nil {
		o.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}

func (o *CommonOptions) runCommandInteractive(interactive bool, name string, args ...string) error {
	e := exec.Command(name, args...)
	e.Stdout = o.Out
	e.Stderr = o.Err
	if interactive {
		e.Stdin = os.Stdin
	}
	err := e.Run()
	if err != nil {
		o.Printf("Error: Command failed  %s %s\n", name, strings.Join(args, " "))
	}
	return err
}
