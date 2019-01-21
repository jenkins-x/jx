package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateJenkinsOptions the options for the create spring command
type CreateJenkinsOptions struct {
	CreateOptions
}

// NewCmdCreateJenkins creates a command object for the "create" command
func NewCmdCreateJenkins(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateJenkinsOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commoncmd.CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "jenkins",
		Short: "Creates a Jenkins resource",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateJenkinsUser(f, in, out, errOut))
	return cmd
}

// Run implements this command
func (o *CreateJenkinsOptions) Run() error {
	return o.Cmd.Help()
}
