package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	delete_addon_gitea_long = templates.LongDesc(`
		Deletes the Gitea addon
`)

	delete_addon_gitea_example = templates.Examples(`
		# Deletes the Gitea addon
		jx delete addon gitea
	`)
)

// DeleteAddonGiteaOptions the options for the create spring command
type DeleteAddonGiteaOptions struct {
	DeleteAddonOptions

	ReleaseName string
}

// NewCmdDeleteAddonGitea defines the command
func NewCmdDeleteAddonGitea(commonOpts *CommonOptions) *cobra.Command {
	options := &DeleteAddonGiteaOptions{
		DeleteAddonOptions: DeleteAddonOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "gitea",
		Short:   "Deletes the Gitea addon",
		Long:    delete_addon_gitea_long,
		Example: delete_addon_gitea_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", "gitea", "The chart release name")
	options.addFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteAddonGiteaOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	err := o.deleteChart(o.ReleaseName, o.Purge)
	if err != nil {
		return err
	}
	return o.deleteGitServer()
}

func (o *DeleteAddonGiteaOptions) deleteGitServer() error {
	options := &DeleteGitServerOptions{
		CommonOptions:       o.CommonOptions,
		IgnoreMissingServer: true,
	}
	options.Args = []string{"gitea"}
	return options.Run()
}
