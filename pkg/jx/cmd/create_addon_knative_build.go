package cmd

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	createAddonKnativeBuildLong = templates.LongDesc(`
		Creates the Knative build addon
`)

	createAddonKnativeBuildExample = templates.Examples(`
		# Create the Knative addon
		jx create addon knative-build
	`)
)

type CreateAddonKnativeBuildOptions struct {
	CreateAddonOptions
}

func NewCmdCreateAddonKnativeBuild(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateAddonKnativeBuildOptions{
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
		Use:     "knative-build",
		Short:   "Create the knative build addon",
		Long:    createAddonKnativeBuildLong,
		Example: createAddonKnativeBuildExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := options.Run()
			CheckErr(err)
		},
	}
	return cmd
}

// Create the addon
func (o *CreateAddonKnativeBuildOptions) Run() error {
	log.Infof("Installing %s addon\n\n", kube.DefaultKnativeBuildReleaseName)

	err := o.CreateAddon(kube.DefaultKnativeBuildReleaseName)
	if err != nil {
		return err
	}

	log.Infof("\n%s installed\n", kube.DefaultKnativeBuildReleaseName)
	log.Infof("To watch a build running use: %s\n", util.ColorInfo("jx logs -k"))
	return nil
}
