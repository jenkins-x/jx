package cmd

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	defaultPipeline  = "release"
	defaultLifecycle = "build"
	defaultMode      = jenkinsfile.CreateStepModePost
)

var (
	createStepLong = templates.LongDesc(`
		Creates a step in the Jenkins X Pipeline
`)

	createStepExample = templates.Examples(`
		# Create a new step in the Jenkins X Pipeline interactively
		jx create step

		# Creates a step on the command line: adding a post step to the release build lifecycle
		jx create step -sh "echo hello world"

		# Creates a step on the command line: adding a pre step to the pullRequest promote lifecycle
		jx create step -p pullRequest -l promote -m pre "echo before promote"
	`)
)

// NewStepDetails configures a new step
type NewStepDetails struct {
	Pipeline  string
	Lifecycle string
	Mode      string
	Step      jenkinsfile.PipelineStep
}

// AddToPipeline adds the step to the given pipeline configuration
func (s *NewStepDetails) AddToPipeline(projectConfig *config.ProjectConfig) error {
	pipelines := projectConfig.GetOrCreatePipelineConfig()
	pipeline, err := pipelines.Pipelines.GetPipeline(s.Pipeline, true)
	if err != nil {
		return err
	}
	lifecycle, err := pipeline.GetLifecycle(s.Lifecycle, true)
	if err != nil {
		return err
	}
	return lifecycle.CreateStep(s.Mode, &s.Step)
}

// CreateStepOptions the options for the create spring command
type CreateStepOptions struct {
	CreateOptions

	Dir            string
	NewStepDetails NewStepDetails
}

// NewCmdCreateStep creates a command object for the "create" command
func NewCmdCreateStep(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateStepOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "step",
		Short:   "Creates a step in the Jenkins X Pipeline",
		Aliases: []string{"steps"},
		Long:    createStepLong,
		Example: createStepExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	step := &options.NewStepDetails
	cmd.Flags().StringVarP(&step.Pipeline, "pipeline", "p", "", "The pipeline kind to add your step. Possible values: "+strings.Join(jenkinsfile.PipelineKinds, ", "))
	cmd.Flags().StringVarP(&step.Lifecycle, "lifecycle", "l", "", "The lifecycle stage to add your step. Possible values: "+strings.Join(jenkinsfile.PipelineLifecycleNames, ", "))
	cmd.Flags().StringVarP(&step.Mode, "mode", "m", "", "The create mode for the new step. Possible values: "+strings.Join(jenkinsfile.CreateStepModes, ", "))
	cmd.Flags().StringVarP(&step.Step.Command, "sh", "c", "", "The command to invoke for the new step")

	return cmd
}

// Run implements the command
func (o *CreateStepOptions) Run() error {
	dir := o.Dir
	var err error
	if dir == "" {
		dir, _, err := o.Git().FindGitConfigDir(o.Dir)
		if err != nil {
			return err
		}
		if dir == "" {
			dir = "."
		}
	}
	projectConfig, fileName, err := config.LoadProjectConfig(dir)
	if err != nil {
		return err
	}

	s := &o.NewStepDetails
	err = o.configureNewStepDetails(s)
	if err != nil {
		return err
	}

	err = s.AddToPipeline(projectConfig)
	if err != nil {
		return err
	}

	err = projectConfig.SaveConfig(fileName)
	if err != nil {
		return err
	}
	log.Infof("Updated Jenkins X Pipeline file: %s\n", util.ColorInfo(fileName))
	return nil

}

func (o *CreateStepOptions) configureNewStepDetails(stepDetails *NewStepDetails) error {
	s := &o.NewStepDetails
	if o.BatchMode {
		if s.Pipeline == "" {
			s.Pipeline = defaultPipeline
		}
		if s.Lifecycle == "" {
			s.Lifecycle = defaultLifecycle
		}
		if s.Mode == "" {
			s.Mode = defaultMode
		}
		if s.Step.Command == "" {
			return util.MissingOption("sh")
		}
		return nil
	}
	var err error

	if s.Pipeline == "" {
		s.Pipeline, err = util.PickNameWithDefault(jenkinsfile.PipelineKinds, "Pick the pipeline kind: ", defaultPipeline, "which kind of pipeline do you want to add a step", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	if s.Lifecycle == "" {
		s.Lifecycle, err = util.PickNameWithDefault(jenkinsfile.PipelineLifecycleNames, "Pick the lifecycle: ", defaultLifecycle, "which lifecycle (stage) do you want to add the step", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	if s.Mode == "" {
		s.Mode, err = util.PickNameWithDefault(jenkinsfile.CreateStepModes, "Pick the create mode: ", defaultMode, "which create mode do you want to use to add the step - pre (before), post (after) or replace?", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	if s.Step.Command == "" {
		prompt := &survey.Input{
			Message: "Command for the new step: ",
			Help:    "The shell command executed inside the container to implement this step",
		}
		err := survey.AskOne(prompt, &s.Step.Command, survey.Required, survey.WithStdio(o.In, o.Out, o.Err))
		if err != nil {
			return err
		}
	}
	return nil
}
