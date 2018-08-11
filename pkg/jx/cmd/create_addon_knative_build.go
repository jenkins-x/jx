package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	createAddonKnativeBuildLong = templates.LongDesc(`
		Creates the Knative Build addon
`)

	createAddonKnativeBuildExample = templates.Examples(`
		# Create the knative addon
		jx create addon knative-build
	`)
)

type CreateAddonKnativeBuildOptions struct {
	CreateAddonOptions
	BackoffLimit int32
	Image        string
}

func NewCmdCreateAddonKnativeBuild(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonKnativeBuildOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "knative-build",
		Short:   "Create the Knative Build addon",
		Aliases: []string{"env"},
		Long:    createAddonKnativeBuildLong,
		Example: createAddonKnativeBuildExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().Int32VarP(&options.BackoffLimit, "backoff-limit", "l", int32(2), "The backoff limit: how many times to retry the job before considering it failed) to run in the Job")
	cmd.Flags().StringVarP(&options.Image, "image", "i", "KnativeBuild/zap2docker-live:latest", "The KnativeBuild image to use to run the ZA Proxy baseline scan")

	return cmd
}

// Create the addon
func (o *CreateAddonKnativeBuildOptions) Run() error {
	log.Info("Installing Knative Build addon\n\n")
	err := o.runCommandVerbose("kubectl", "apply", "-f", "https://storage.googleapis.com/knative-releases/build/latest/release.yaml")

	if err != nil {
		return err
	}

	log.Infof("\nKnative Build installed\n")
	log.Infof("To watch a build running use: %s\n", util.ColorInfo("jx logs -k"))
	return nil
}
