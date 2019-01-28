package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	defaultFlaggerNamespace   = "istio-system"
	defaultFlaggerReleaseName = "flagger"
	defaultFlaggerVersion     = ""
	defaultFlaggerRepo        = "https://flagger.app"
)

var (
	createAddonFlaggerLong = templates.LongDesc(`
		Creates the Flagger addon
`)

	createAddonFlaggerExample = templates.Examples(`
		# Create the Flagger addon
		jx create addon flagger
	`)
)

type CreateAddonFlaggerOptions struct {
	CreateAddonOptions
	Chart string
}

func NewCmdCreateAddonFlagger(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateAddonFlaggerOptions{
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
		Use:     "flagger",
		Short:   "Create the Flagger addon for Canary deployments",
		Long:    createAddonFlaggerLong,
		Example: createAddonFlaggerExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultFlaggerNamespace, defaultFlaggerReleaseName, defaultFlaggerVersion)

	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartFlagger, "The name of the chart to use")
	return cmd
}

// Create the addon
func (o *CreateAddonFlaggerOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that Helm is present")
	}

	values := []string{}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	err = o.addHelmRepoIfMissing(defaultFlaggerRepo, "flagger", "", "")
	if err != nil {
		return fmt.Errorf("Flagger deployment failed: %v", err)
	}
	err = o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values, nil, "")
	if err != nil {
		return fmt.Errorf("Flagger deployment failed: %v", err)
	}
	return nil
}
