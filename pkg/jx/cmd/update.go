package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/update"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// Update contains the command line options
type UpdateOptions struct {
	*opts.CommonOptions

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
func NewCmdUpdate(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &UpdateOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updates an existing resource",
		Long:  update_long,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateCluster(commonOpts))
	cmd.AddCommand(update.NewCmdUpdateWebhooks(commonOpts))

	return cmd
}

// Run implements this command
func (o *UpdateOptions) Run() error {
	return o.Cmd.Help()
}
