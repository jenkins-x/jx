package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/kube"
	"io"
	"strings"

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
	username string
	token    string
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
	cmd.Flags().StringVarP(&options.username, "username", "u", "", "The pipeline bot username")
	cmd.Flags().StringVarP(&options.token, "token", "t", "", "The pipeline bot token")
	return cmd
}

// Create the addon
func (o *CreateAddonKnativeBuildOptions) Run() error {
	if o.token == "" {
		return fmt.Errorf("no pipeline git token provided")
	}
	log.Infof("Installing %s addon\n\n", kube.DefaultKnativeBuildReleaseName)

	o.SetValues = strings.Join([]string{"build.auth.git.username=" + o.username, "build.auth.git.password=" + o.token}, ",")

	err := o.CreateAddon(kube.DefaultKnativeBuildReleaseName)
	if err != nil {
		return err
	}

	log.Infof("\n%s installed\n", kube.DefaultKnativeBuildReleaseName)
	log.Infof("To watch a build running use: %s\n", util.ColorInfo("jx logs -k"))
	return nil
}
