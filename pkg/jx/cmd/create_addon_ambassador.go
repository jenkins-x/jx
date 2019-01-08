package cmd

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	defaultAmbassadorReleaseName = "ambassador"
	ambassadorRepoName           = "datawire"
	ambassadorRepoUrl            = "https://www.getambassador.io"
	defaultAmbassadorVersion     = ""
)

var (
	createAddonAmbassadorLong = templates.LongDesc(`
		Creates the ambassador addon for smart load balancing on kubernetes
`)

	createAddonAmbassadorExample = templates.Examples(`
		# Create the ambassador addon 
		jx create addon ambassador

		# Create the ambassador addon in a custom namespace
		jx create addon ambassador -n mynamespace
	`)
)

// CreateAddonAmbassadorOptions the options for the create spring command
type CreateAddonAmbassadorOptions struct {
	CreateAddonOptions

	Chart string
}

// NewCmdCreateAddonAmbassador creates a command object for the "create" command
func NewCmdCreateAddonAmbassador(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateAddonAmbassadorOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "ambassador",
		Short:   "Create an ambassador addon",
		Aliases: []string{"env"},
		Long:    createAddonAmbassadorLong,
		Example: createAddonAmbassadorExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, "", defaultAmbassadorReleaseName, defaultAmbassadorVersion)

	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartAmbassador, "The name of the chart to use")
	return cmd
}

// Run implements the command
func (o *CreateAddonAmbassadorOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	err := o.addHelmRepoIfMissing(ambassadorRepoUrl, ambassadorRepoName, "", "")
	if err != nil {
		return err
	}

	err = o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that helm is present")
	}

	values := strings.Split(o.SetValues, ",")
	err = o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values, nil, "")
	if err != nil {
		return err
	}
	return nil
}
