package create

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/create/options"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
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

func NewCmdCreateAddonKnativeBuild(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateAddonKnativeBuildOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: options.CreateOptions{
				CommonOptions: commonOpts,
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
			helper.CheckErr(err)
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
	log.Logger().Infof("Installing %s addon\n", kube.DefaultKnativeBuildReleaseName)

	o.SetValues = strings.Join([]string{"build.auth.git.username=" + o.username, "build.auth.git.password=" + o.token}, ",")

	if o.Namespace == "" {
		_, currentNamespace, err := o.KubeClientAndNamespace()
		if err != nil {
			return err
		}
		o.Namespace = currentNamespace
	}

	err := o.CreateAddon(kube.DefaultKnativeBuildReleaseName)
	if err != nil {
		return err
	}

	log.Logger().Infof("\n%s installed", kube.DefaultKnativeBuildReleaseName)
	log.Logger().Infof("To watch a build running use: %s", util.ColorInfo("jx logs -k"))
	return nil
}
