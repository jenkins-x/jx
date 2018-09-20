package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepNextBuildNumberOptions contains the command line flags
type StepNextBuildNumberOptions struct {
	StepOptions

	Owner      string
	Repository string
	Branch     string
}

var (
	StepNextBuildNumberLong = templates.LongDesc(`
		TGenerates the next build unique number for a pipeline
`)

	StepNextBuildNumberExample = templates.Examples(`
		jx step next-buildnumber 
`)
)

func NewCmdStepNextBuildNumber(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepNextBuildNumberOptions{
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
		Use:     "next-buildnumber",
		Short:   "Generates the next build unique number for a pipeline.",
		Long:    StepNextBuildNumberLong,
		Example: StepNextBuildNumberExample,
		Aliases: []string{"next-buildno"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Owner, optionOwner, "o", "", "The git repository owner")
	cmd.Flags().StringVarP(&options.Repository, optionRepo, "r", "", "The git repository name")
	cmd.Flags().StringVarP(&options.Branch, "branch", "b", "master", "The git branch")
	return cmd
}

func (o *StepNextBuildNumberOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	owner := o.Owner
	repository := o.Repository
	branch := o.Branch

	if owner == "" {
		return util.MissingOption(optionOwner)
	}
	if repository == "" {
		return util.MissingOption(optionRepo)
	}
	build, _, err := kube.GenerateBuildNumber(activities, owner, repository, branch)
	if err != nil {
		return err
	}
	log.Infof("%s\n", build)
	return nil
}
