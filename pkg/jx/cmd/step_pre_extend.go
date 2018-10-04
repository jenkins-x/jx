package cmd

import (
	"github.com/stoewer/go-strcase"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepPreBuildOptions contains the command line flags
type StepPreExtendOptions struct {
	StepOptions
}

var (
	StepPreExtendLong = templates.LongDesc(`
		This pipeline step adds any extensions configured for this pipeline
`)

	StepPreExtendExample = templates.Examples(`
		jx step pre extend
`)
)

func NewCmdStepPreExtend(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepPreExtendOptions{
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
		Use:     "extend",
		Short:   "Adds any extensions configured for this pipeline",
		Long:    StepPreExtendLong,
		Example: StepPreExtendExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	return cmd
}

func (o *StepPreExtendOptions) Run() error {

	f := o.Factory
	client, ns, err := f.CreateJXClient()
	if err != nil {
		return errors.Wrap(err, "cannot create the JX client")
	}

	extensionsClient := client.JenkinsV1().Extensions(ns)
	repoExtensions, err := (&kube.ExtensionsConfig{}).LoadFromFile()
	if err != nil {
		return err
	}
	availableExtensions, err := extensionsClient.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	availableExtensionsNames := []string{}
	for _, ae := range availableExtensions.Items {
		availableExtensionsNames = append(availableExtensionsNames, ae.Name)
	}

	if len(repoExtensions.Extensions) > 0 {

		apisClient, err := o.CreateApiExtensionsClient()
		if err != nil {
			return err
		}
		err = kube.RegisterPipelineActivityCRD(apisClient)
		if err != nil {
			return err
		}

		activities := client.JenkinsV1().PipelineActivities(ns)
		if err != nil {
			return err
		}
		gitInfo, err := o.FindGitInfo("")
		appName := ""
		if gitInfo != nil {
			appName = gitInfo.Name
		}
		pipeline := ""
		build := o.getBuildNumber()
		pipeline, build = o.getPipelineName(gitInfo, pipeline, build, appName)
		if pipeline != "" && build != "" {
			name := kube.ToValidName(pipeline + "-" + build)
			key := &kube.PromoteStepActivityKey{
				PipelineActivityKey: kube.PipelineActivityKey{
					Name:     name,
					Pipeline: pipeline,
					Build:    build,
				},
			}
			a, _, err := key.GetOrCreate(activities)
			if err != nil {
				return err
			}
			for k, v := range repoExtensions.Extensions {
				e, err := extensionsClient.Get(strcase.KebabCase(k), metav1.GetOptions{})
				name := strcase.KebabCase(k)
				if err != nil {
					// Extension can't be found
					log.Infof("Extension %s applied but cannot be found in this Jenkins X installation. Available extensions are %s", name, availableExtensionsNames)
				} else {
					if o.Verbose {
						log.Infof("Adding extension %s", util.ColorInfo(name))
					}

					if o.Contains(e.Spec.When, jenkinsv1.ExtensionWhenPost) || len(e.Spec.When) == 0 {

						if a.Spec.PostExtensions == nil {
							a.Spec.PostExtensions = map[string]jenkinsv1.ExecutableExtension{}
						}
						ext, envVarsFormatted, err := e.Spec.ToExecutable(v.Parameters)
						if err != nil {
							return err
						}
						a.Spec.PostExtensions[e.Name] = ext
						log.Infof("Adding Extension %s version %s to pipeline with environment variables [ %s ]\n", util.ColorInfo(e.Spec.Name), util.ColorInfo(e.Spec.Version), util.ColorInfo(envVarsFormatted))
					}
				}
			}
			a, err = activities.Update(a)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *StepPreExtendOptions) Contains(whens []jenkinsv1.ExtensionWhen, when jenkinsv1.ExtensionWhen) bool {
	for _, w := range whens {
		if when == w {
			return true
		}
	}
	return false
}
