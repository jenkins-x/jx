package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// Update contacommonOptss the command lcommonOptse options
type UpdateOptions struct {
	*CommonOptions

	DisableImport bool
	OutDir        string
}

var (
	update_resources = `Valid resource types commonOptsclude:

	* cluster
	`

	update_long = templates.LongDesc(`
		Updates an existcommonOptsg resource.

		` + update_resources + `
`)
)

// NewCmdUpdate creates a command object for the "update" command
func NewCmdUpdate(commonOpts *CommonOptions) *cobra.Command {
	options := &UpdateOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updates an existcommonOptsg resource",
		Long:  update_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateCluster(commonOpts))
	cmd.AddCommand(NewCmdUpdateWebhooks(commonOpts))

	return cmd
}

// Run implements this command
func (o *UpdateOptions) Run() error {
	return o.Cmd.Help()
}
