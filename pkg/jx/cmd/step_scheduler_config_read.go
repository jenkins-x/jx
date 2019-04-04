package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/jenkinsfile"

	"github.com/jenkins-x/jx/pkg/environments"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StepSchedulerConfigReadOptions contains the command line flags
type StepSchedulerConfigReadOptions struct {
	StepOptions
	Namespace string
	Dir       string
	Contexts  []string

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback environments.ConfigureGitFn
}

var (
	stepSchedulerConfigReadLong = templates.LongDesc(`
		This pipeline step command reads the pipeline configuration from a jenkins-x.yml.
		The contexts to read the jenkins-x.yml for can be specified.
		
		This step is run automatically by every pipeline. 

		jx step scheduler config apply causes the read configuration to be applied to prow (or added to git).

`)
	stepSchedulerConfigReadExample = templates.Examples(`
	
	jx step scheduler config read --context integration --context lint --context bdd
`)
)

// NewCmdStepSchedulerConfigRead Steps a command object for the "step" command
func NewCmdStepSchedulerConfigRead(commonOpts *CommonOptions) *cobra.Command {
	options := &StepSchedulerConfigReadOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "read",
		Short:   "scheduler config read",
		Long:    stepSchedulerConfigReadLong,
		Example: stepSchedulerConfigReadExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "",
		"The Kubernetes namespace to read the scheduler into")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to look for jenkins-x.yml")
	cmd.Flags().StringArrayVarP(&options.Contexts, "context", "", []string{""}, "The contexts to read")
	return cmd
}

// Run implements this command
func (o *StepSchedulerConfigReadOptions) Run() error {
	if o.Dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "getting working directory")
		}
		o.Dir = dir
	}
	for _, context := range o.Contexts {
		filename := "jenkins-x.yml"
		if context != "" {
			filename = fmt.Sprintf("jenkins-x-%s.yml", context)
		}
		path := filepath.Join(o.Dir, filename)
		if _, err := os.Stat(path); err != nil {
			return errors.Wrapf(err, "loading file %s", path)
		}
		resolver := CreateImport
		jenkinsfile.LoadPipelineConfig(path)
	}

	return nil
}
