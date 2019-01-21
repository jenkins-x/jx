package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// Update contains the command line options
type UpdateOptions struct {
	commoncmd.CommonOptions

	DisableImport bool
	OutDir        string
}

var (
	update_resources = `Valid resource types include:

	* cluster
	`

	update_long = templates.LongDesc(`
		Updates an existing resource.

		` + update_resources + `
`)
)

// NewCmdUpdate creates a command object for the "update" command
func NewCmdUpdate(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpdateOptions{
		CommonOptions: commoncmd.CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updates an existing resource",
		Long:  update_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateCluster(f, in, out, errOut))
	cmd.AddCommand(NewCmdUpdateWebhooks(f, in, out, errOut))

	return cmd
}

// Run implements this command
func (o *UpdateOptions) Run() error {
	return o.Cmd.Help()
}
