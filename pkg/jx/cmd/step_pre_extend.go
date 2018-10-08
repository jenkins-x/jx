package cmd

import (
	"fmt"
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

const extensionsConfigDefaultFile = "jenkins-x-extensions.yaml"

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

	apisClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterExtensionCRD(apisClient)
	if err != nil {
		return err
	}

	extensionsClient := client.JenkinsV1().Extensions(ns)
	repoExtensions, err := (&jenkinsv1.ExtensionConfigList{}).LoadFromFile(extensionsConfigDefaultFile)
	if err != nil {
		return err
	}
	availableExtensions, err := extensionsClient.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	availableExtensionsNames := []string{}
	availableExtensionsUUIDLookup := make(map[string]jenkinsv1.ExtensionSpec, 0)
	for _, ae := range availableExtensions.Items {
		availableExtensionsNames = append(availableExtensionsNames, ae.Name)
		availableExtensionsUUIDLookup[ae.Spec.UUID] = ae.Spec
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
			for _, v := range repoExtensions.Extensions {
				e, err := extensionsClient.Get(v.FullyQualifiedKebabName(), metav1.GetOptions{})
				if err != nil {
					// Extension can't be found
					log.Infof("Extension %s applied but cannot be found in this Jenkins X installation. Available extensions are %s\n", util.ColorInfo(fmt.Sprintf("%s", v.FullyQualifiedName())), util.ColorInfo(availableExtensionsNames))
				} else {
					if o.Verbose {
						log.Infof("Adding extension %s", util.ColorInfo(name))
					}
					if len(e.Spec.Children) > 0 {
						log.Infof("Adding Extension %s version %s to pipeline\n", util.ColorInfo(e.Spec.FullyQualifiedName()), util.ColorInfo(e.Spec.Version))
						for _, childUUID := range e.Spec.Children {
							if child, ok := availableExtensionsUUIDLookup[childUUID]; ok {
								err = o.AddPipelineExtension(&a.Spec, child, v.Parameters, true)
								if err != nil {
									return err
								}
							}
						}
					} else {
						err = o.AddPipelineExtension(&a.Spec, e.Spec, v.Parameters, false)
						if err != nil {
							return err
						}
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

func (o *StepPreExtendOptions) AddPipelineExtension(a *jenkinsv1.PipelineActivitySpec, e jenkinsv1.ExtensionSpec, parameters []jenkinsv1.ExtensionParameterValue, child bool) (err error) {
	if e.IsPost() {

		if a.PostExtensions == nil {
			a.PostExtensions = make([]jenkinsv1.ExtensionExecution, 0)
		}
		ext, envVarsFormatted, err := e.ToExecutable(parameters)
		if err != nil {
			return err
		}
		a.PostExtensions = append(a.PostExtensions, ext)
		envVarsStr := ""
		if len(envVarsFormatted) > 0 {
			envVarsStr = fmt.Sprintf("with environment variables [ %s ]", util.ColorInfo(envVarsFormatted))
		}
		if child {
			log.Infof(" └ %s version %s %s\n", util.ColorInfo(e.FullyQualifiedName()), util.ColorInfo(e.Version), envVarsStr)
		} else {
			log.Infof("Adding Extension %s version %s to pipeline %s\n", util.ColorInfo(e.FullyQualifiedName()), util.ColorInfo(e.Version), envVarsStr)
		}
	}
	return nil
}
