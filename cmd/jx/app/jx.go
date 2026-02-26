// +build !windows

package app

import (
	"os"

	cmd2 "github.com/jenkins-x/jx/v2/pkg/cmd"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
)

// Run runs the command, if args are not nil they will be set on the command
func Run(args []string) error {
	cmd := cmd2.NewJXCommand(clients.NewFactory(), os.Stdin, os.Stdout, os.Stderr, nil)
	if args != nil {
		args = args[1:]
		cmd.SetArgs(args)
	}
	return cmd.Execute()
}
