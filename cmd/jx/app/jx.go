package app

import (
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
)

// Run runs the command
func Run() error {
	/*
		logs.InitLogs()
		defer logs.FlushLogs()
	*/

	cmd := cmd.NewJXCommand(clients.NewFactory(), os.Stdin, os.Stdout, os.Stderr, nil)
	return cmd.Execute()
}
