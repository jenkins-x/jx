package helm

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"os"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
)

// StepHelmBuildOptions contains the command line flags
type StepHelmBuildOptions struct {
	StepHelmOptions

	recursive bool
}

var (
	StepHelmBuildLong = templates.LongDesc(`
		Builds the helm chart in a given directory.

		This step is usually used to validate any GitOps Pull Requests.
`)

	StepHelmBuildExample = templates.Examples(`
		# builds the helm chart in the env directory
		jx step helm build --dir env

`)
)

func NewCmdStepHelmBuild(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepHelmBuildOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: opts.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "build",
		Short:   "Builds the helm chart in a given directory and validate the build completes",
		Aliases: []string{""},
		Long:    StepHelmBuildLong,
		Example: StepHelmBuildExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.addStepHelmFlags(cmd)

	cmd.Flags().BoolVarP(&options.recursive, "recursive", "r", false, "Build recursively the dependent charts")

	return cmd
}

func (o *StepHelmBuildOptions) Run() error {
	_, _, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}

	dir := o.Dir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	valuesFiles, err := o.discoverValuesFiles(dir)
	if err != nil {
		return err
	}

	if o.recursive {
		return o.HelmInitRecursiveDependencyBuild(dir, o.DefaultReleaseCharts(), valuesFiles)
	}
	_, err = o.HelmInitDependencyBuild(dir, o.DefaultReleaseCharts(), valuesFiles)
	return err
}
