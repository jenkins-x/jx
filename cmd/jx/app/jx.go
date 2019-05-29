// +build !windows

package app

import (
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
)

// Run runs the command
func Run(args []string) error {
	if len(args) > 0 {
		args = args[1:]
	}
	cmd := cmd.NewJXCommand(clients.NewFactory(), os.Stdin, os.Stdout, os.Stderr, nil)
	return cmd.Execute()
}
