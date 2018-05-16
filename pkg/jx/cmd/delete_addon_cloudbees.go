package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	deleteAddonCloudBeesLong = templates.LongDesc(`
		Deletes the CloudBees addon
`)

	deleteAddonCloudBeesExample = templates.Examples(`
		# Deletes the CloudBees addon
		jx delete addon cloudbees
	`)
)

// DeleteAddonGiteaOptions the options for the create spring command
type DeleteAddonCDXOptions struct {
	DeleteAddonOptions

	ReleaseName string
}

// NewCmdDeleteAddonCloudBees defines the command
func NewCmdDeleteAddonCloudBees(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
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
		Use:     "cloudbees",
		Short:   "Deletes the CloudBees app for Kubernetes addon",
		Aliases: []string{"cloudbee", "cb", "cdx"},
		Long:    deleteAddonCloudBeesLong,
		Example: deleteAddonCloudBeesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", defaultCloudBeesReleaseName, "The chart release name")
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
