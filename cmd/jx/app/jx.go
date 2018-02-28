package app

import (
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// Run runs the command
func Run() error {
	/*
		logs.InitLogs()
		defer logs.FlushLogs()
	*/

	cmd := cmd.NewJXCommand(cmdutil.NewFactory(), os.Stdin, os.Stdout, os.Stderr)
	return cmd.Execute()
}
