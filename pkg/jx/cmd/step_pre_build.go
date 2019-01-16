package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepPreBuildOptions contains the command line flags
type StepPreBuildOptions struct {
	StepOptions

	Image string
}

var (
	StepPreBuildLong = templates.LongDesc(`
		This pipeline step performs pre build actions such as ensuring that a Docker registry is available in the cloud
`)

	StepPreBuildExample = templates.Examples(`
		jx step pre build ${DOCKER_REGISTRY}/someorg/myapp
`)
)

func NewCmdStepPreBuild(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepPreBuildOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "build",
		Short:   "Performs actions before a build happens in a pipeline",
		Long:    StepPreBuildLong,
		Example: StepPreBuildExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Image, optionImage, "i", "", "The image name that is about to be built")
	return cmd
}

func (o *StepPreBuildOptions) Run() error {
	imageName := o.Image
	if imageName == "" {
		args := o.Args
		if len(args) == 0 {
			return util.MissingOption(optionImage)
		} else {
			imageName = args[0]
		}
	}
	paths := strings.Split(imageName, "/")
	l := len(paths)
	if l > 2 {
		dockerRegistry := paths[0]
		orgName := paths[l-2]
		appName := paths[l-1]

		log.Infof("Docker registry host: %s app name %s/%s\n", util.ColorInfo(dockerRegistry), util.ColorInfo(orgName), util.ColorInfo(appName))

		kube, err := o.KubeClient()
		if err != nil {
			return err
		}
		region, _ := amazon.ReadRegion(kube, o.currentNamespace)
		if strings.HasSuffix(dockerRegistry, ".amazonaws.com") && strings.Index(dockerRegistry, ".ecr.") > 0 {
			return amazon.LazyCreateRegistry(kube, o.currentNamespace, region, dockerRegistry, orgName, appName)
		}
	}
	return nil
}
