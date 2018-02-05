package cmd

import (
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// CreateJenkinsOptions the options for the create spring command
type CreateJenkinsOptions struct {
	CreateOptions
}

// NewCmdCreateJenkins creates a command object for the "create" command
func NewCmdCreateJenkins(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateJenkinsOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "jenkins",
		Short: "Creates a jenkins resource",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateJenkinsUser(f, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateJenkinsOptions) Run() error {
	return o.Cmd.Help()
}
