package create

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/spf13/cobra"
)

// CreateGitOptions the options for the create spring command
type CreateGitOptions struct {
	CreateOptions
}

// NewCmdCreateGit creates a command object for the "create" command
func NewCmdCreateGit(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateGitOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "git",
		Short:   "Creates a Git resource",
		Aliases: []string{"scm"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateGitServer(commonOpts))
	cmd.AddCommand(NewCmdCreateGitToken(commonOpts))
	cmd.AddCommand(NewCmdCreateGitUser(commonOpts))
	return cmd
}

// Run implements this command
func (o *CreateGitOptions) Run() error {
	return o.Cmd.Help()
}
