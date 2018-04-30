package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	delete_addon_cdx_long = templates.LongDesc(`
		Deletes the cdx addon
`)

	delete_addon_cdx_example = templates.Examples(`
		# Deletes the gitea addon
		jx delete addon cdx
	`)
)

// DeleteAddonGiteaOptions the options for the create spring command
type DeleteAddonCDXOptions struct {
	DeleteAddonOptions

	ReleaseName string
}

// NewCmdDeleteAddonGitea defines the command
func NewCmdDeleteAddonCDX(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteAddonGiteaOptions{
		DeleteAddonOptions: DeleteAddonOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "cdx",
		Short:   "Deletes the cdx addon",
		Long:    delete_addon_cdx_long,
		Example: delete_addon_cdx_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", "cdx", "The chart release name")
	options.addFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteAddonCDXOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	err := o.deleteChart(o.ReleaseName, o.Purge)
	if err != nil {
		return err
	}

	return nil
}
