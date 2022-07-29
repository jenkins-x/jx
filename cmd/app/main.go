//nolint:gofmt,goimports
//go:build !windows

package app

import "github.com/jenkins-x/jx/pkg/cmd"

// Run runs the command, if args are not nil they will be set on the command
func Run(args []string) error {
	c := cmd.Main(args)
	if args != nil {
		args = args[1:]
		c.SetArgs(args)
	}
	return c.Execute()
}
