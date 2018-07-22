package cmd

import (
	"io"

	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"os"
)

// StepHelmBuildOptions contains the command line flags
type StepHelmBuildOptions struct {
	StepHelmOptions
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

func NewCmdStepHelmBuild(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepHelmBuildOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
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
			CheckErr(err)
		},
	}

	options.addStepHelmFlags(cmd)
	return cmd
}

func (o *StepHelmBuildOptions) Run() error {

	_, _, err := o.KubeClient()
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

	// if we're in a prow job we need to clone and change dir to find the Helm Chart.yaml
	if os.Getenv(PROW_JOB_ID) != "" {
		dir, err = o.cloneProwPullRequest(dir, o.GitProvider)
		if err != nil {
			return fmt.Errorf("failed to clone pull request: %v", err)
		}
	}
	_, err = o.helmInitDependencyBuild(dir, o.defaultReleaseCharts())
	return err
}
