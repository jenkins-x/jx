package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// Update contains the command line options
type UpdateOptions struct {
	CommonOptions

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
func NewCmdUpdate(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &UpdateOptions{
		CommonOptions: CommonOptions{
			Factory: f,
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
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateCluster(f, out, errOut))

	return cmd
}

// Run implements this command
func (o *UpdateOptions) Run() error {
	return o.Cmd.Help()
}
