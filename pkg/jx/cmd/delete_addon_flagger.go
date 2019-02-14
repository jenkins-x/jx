package cmd

import (
	"github.com/pkg/errors"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	deleteAddonFlaggerLong = templates.LongDesc(`
		Deletes the Flagger addon
`)

	deleteAddonFlaggerExample = templates.Examples(`
		# Deletes the Flagger addon
		jx delete addon flagger
	`)
)

// DeleteAddonFlaggerOptions the options for the create spring command
type DeleteAddonFlaggerOptions struct {
	DeleteAddonOptions

	ReleaseName string
}

// NewCmdDeleteAddonFlagger defines the command
func NewCmdDeleteAddonFlagger(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteAddonFlaggerOptions{
		DeleteAddonOptions: DeleteAddonOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "flagger",
		Short:   "Deletes the Flagger addon",
		Long:    deleteAddonFlaggerLong,
		Example: deleteAddonFlaggerExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", "flagger", "The chart release name")
	options.addFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteAddonFlaggerOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	err := o.deleteChart(o.ReleaseName, o.Purge)
	if err != nil {
		return errors.Wrap(err, "Failed to delete Flagger chart")
	}
	err = o.deleteChart(o.ReleaseName+"-grafana", o.Purge)
	if err != nil {
		return errors.Wrap(err, "Failed to delete Flagger Grafana chart")
	}
	return err
}
