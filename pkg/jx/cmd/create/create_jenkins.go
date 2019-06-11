package create

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/spf13/cobra"
)

// CreateJenkinsOptions the options for the create spring command
type CreateJenkinsOptions struct {
	CreateOptions
}

// NewCmdCreateJenkins creates a command object for the "create" command
func NewCmdCreateJenkins(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateJenkinsOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "jenkins",
		Short: "Creates a Jenkins resource",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateJenkinsUser(commonOpts))
	return cmd
}

// Run implements this command
func (o *CreateJenkinsOptions) Run() error {
	return o.Cmd.Help()
}
