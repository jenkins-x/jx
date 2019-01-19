package cmd

import (
	"fmt"
	"io"
	"strings"

	jenkinsv1client "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/extensions"

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
	// This will cause o.devNamespace to be populated
	client, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	apisClient, err := o.ApiExtensionsClient()
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

		apisClient, err := o.ApiExtensionsClient()
		if err != nil {
			return err
		}
		err = kube.RegisterPipelineActivityCRD(apisClient)
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
			a, _, err := key.GetOrCreate(client,ns)
			if err != nil {
				return err
			}
			for _, v := range repoExtensions.Extensions {
				e, err := extensionsClient.Get(v.FullyQualifiedKebabName(), metav1.GetOptions{})
				if err != nil {
					// Extension can't be found
					log.Infof("Extension %s applied but cannot be found in this Jenkins X installation. Available extensions are %s\n", util.ColorInfo(fmt.Sprintf("%s", v.FullyQualifiedName())), util.ColorInfo(availableExtensionsNames))
				} else {
					result, err := o.walk(&e.Spec, availableExtensionsUUIDLookup, v.Parameters, 0, client.JenkinsV1().Extensions(ns))
					if err != nil {
						return err
					}
					a.Spec.PostExtensions = result
				}
			}
			a, err = client.JenkinsV1().PipelineActivities(ns).Update(a)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *StepPreExtendOptions) walk(extension *jenkinsv1.ExtensionSpec, lookup map[string]jenkinsv1.ExtensionSpec, parameters []jenkinsv1.ExtensionParameterValue, depth int, exts jenkinsv1client.ExtensionInterface) (result []jenkinsv1.ExtensionExecution, err error) {
	result = make([]jenkinsv1.ExtensionExecution, 0)
	if len(extension.Children) > 0 {
		if depth > 0 {
			indent := ((depth - 1) * 2) + 7
			log.Infof("%s└ %s version %s\n", strings.Repeat(" ", indent), util.ColorInfo(extension.FullyQualifiedName()), util.ColorInfo(extension.Version))
		} else {
			log.Infof("Adding %s version %s to pipeline\n", util.ColorInfo(extension.FullyQualifiedName()), util.ColorInfo(extension.Version))
		}
		for _, childRef := range extension.Children {
			if child, ok := lookup[childRef]; ok {
				children, err := o.walk(&child, lookup, parameters, depth+1, exts)
				if err != nil {
					return result, err
				}
				result = append(result, children...)
			} else {
				errors.New(fmt.Sprintf("Unable to locate extension %s", childRef))
			}
		}
	} else {
		if extension.IsPost() {
			ext, envVarsFormatted, err := extensions.ToExecutable(extension, parameters, o.devNamespace, exts)
			if err != nil {
				return result, err
			}
			envVarsStr := ""
			if len(envVarsFormatted) > 0 {
				envVarsStr = fmt.Sprintf("with environment variables [ %s ]", util.ColorInfo(envVarsFormatted))
			}
			if depth > 0 {
				indent := ((depth - 1) * 2) + 7
				log.Infof("%s└ %s version %s %s\n", strings.Repeat(" ", indent), util.ColorInfo(extension.FullyQualifiedName()), util.ColorInfo(extension.Version), envVarsStr)
			} else {
				log.Infof("Adding %s version %s to pipeline %s\n", util.ColorInfo(extension.FullyQualifiedName()), util.ColorInfo(extension.Version), envVarsStr)
			}
			result = append(result, ext)
		}
	}
	return result, nil
}
