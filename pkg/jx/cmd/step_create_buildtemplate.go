package cmd

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jenkinsfile/gitresolver"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	buildapi "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultContainerName = "maven"
)

var (
	createBuildTemplateLong = templates.LongDesc(`
		Creates a Knative build resource for a project
`)

	createBuildTemplateExample = templates.Examples(`
		# create a Knative build and render to the console
		jx step create buildtemplate

		# create a Knative build
		jx step create buildtemplate -o mybuild.yaml

			`)
)

// StepCreateBuildTemplateOptions contains the command line flags
type StepCreateBuildTemplateOptions struct {
	StepOptions

	Dir          string
	OutputDir    string
	BuildPackURL string
	BuildPackRef string

	PodTemplates        map[string]*corev1.Pod
	MissingPodTemplates map[string]bool
}

// NewCmdStepCreateBuildTemplate Creates a new Command object
func NewCmdStepCreateBuildTemplate(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepCreateBuildTemplateOptions{
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
		Use:     "buildtemplate",
		Short:   "Creates a Knative build templates based on the current build pack",
		Long:    createBuildTemplateLong,
		Example: createBuildTemplateExample,
		Aliases: []string{"bt"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to query to find the projects .git directory")
	cmd.Flags().StringVarP(&options.OutputDir, "output-dir", "o", "jx-build-templates", "The directory where the generated build yaml files will be output to")
	cmd.Flags().StringVarP(&options.BuildPackURL, "url", "u", "", "The URL for the build pack Git repository")
	cmd.Flags().StringVarP(&options.BuildPackRef, "ref", "r", "", "The Git reference (branch,tag,sha) in the Git repository to use")
	return cmd
}

// Run implements this command
func (o *StepCreateBuildTemplateOptions) Run() error {
	if o.BuildPackURL == "" || o.BuildPackRef == "" {
		settings, err := o.TeamSettings()
		if err != nil {
			return err
		}
		if o.BuildPackURL == "" {
			o.BuildPackURL = settings.BuildPackURL
		}
		if o.BuildPackRef == "" {
			o.BuildPackRef = settings.BuildPackRef
		}
	}
	if o.BuildPackURL == "" {
		return util.MissingOption("url")
	}
	if o.BuildPackRef == "" {
		return util.MissingOption("ref")
	}
	if o.OutputDir == "" {
		return util.MissingOption("output-dir")
	}

	err := o.loadPodTemplates()
	if err != nil {
		return err
	}
	o.MissingPodTemplates = map[string]bool{}

	err = os.MkdirAll(o.OutputDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create output dir %s", o.OutputDir)
	}

	packDir, err := gitresolver.InitBuildPack(o.Git(), o.BuildPackURL, o.BuildPackRef)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(packDir)
	if err != nil {
		return err
	}
	resolver, err := gitresolver.CreateResolver(packDir, o.Git())
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			pipelineFile := filepath.Join(packDir, name, jenkinsfile.PipelineConfigFileName)
			exists, err := util.FileExists(pipelineFile)
			if err != nil {
				return err
			}
			if exists {
				pipelineConfig, err := jenkinsfile.LoadPipelineConfig(pipelineFile, resolver, true, false)
				if err != nil {
					return err
				}
				err = o.generateBuildTemplate(name, pipelineConfig)
				if err != nil {
					return err
				}
			} else {
				log.Infof("No pipeline YAML file for %s\n", pipelineFile)
			}
		}
	}
	for k := range o.MissingPodTemplates {
		log.Warnf("Missing pod template for container %s\n", k)
	}
	log.Infof("generated build packs at %s\n", util.ColorInfo(o.OutputDir))
	return err
}

func (o *StepCreateBuildTemplateOptions) loadPodTemplates() error {
	o.PodTemplates = map[string]*corev1.Pod{}

	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	configMapName := kube.ConfigMapJenkinsPodTemplates
	cm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for k, v := range cm.Data {
		pod := &corev1.Pod{}
		if v != "" {
			err := yaml.Unmarshal([]byte(v), pod)
			if err != nil {
				return err
			}
			o.PodTemplates[k] = pod
		}
	}
	return nil
}

func (o *StepCreateBuildTemplateOptions) generateBuildTemplate(name string, pipelineConfig *jenkinsfile.PipelineConfig) error {
	pipelines := pipelineConfig.Pipelines
	err := o.generatePipeline(name, pipelineConfig, pipelines.Release, "release")
	if err != nil {
		return err
	}
	err = o.generatePipeline(name, pipelineConfig, pipelines.PullRequest, "pullrequest")
	if err != nil {
		return err
	}
	return o.generatePipeline(name, pipelineConfig, pipelines.Feature, "feature")
}

func (o *StepCreateBuildTemplateOptions) generatePipeline(languageName string, pipelineConfig *jenkinsfile.PipelineConfig, lifecycles *jenkinsfile.PipelineLifecycles, templateKind string) error {
	if lifecycles == nil {
		return nil
	}

	container := pipelineConfig.Agent.Container
	dir := "/workspace"

	steps := []corev1.Container{}
	for _, l := range lifecycles.All() {
		if l == nil {
			continue
		}
		for _, s := range l.Steps {
			ss, err := o.createSteps(languageName, pipelineConfig, templateKind, s, container, dir)
			if err != nil {
				return err
			}
			steps = append(steps, ss...)
		}
	}
	name := "jx-buildtemplate-" + languageName + "-" + templateKind
	build := &buildapi.BuildTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "build.knative.dev/v1alpha1",
			Kind:       "BuildTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kube.ToValidName(name),
		},
		Spec: buildapi.BuildTemplateSpec{
			Steps: steps,
		},
	}
	fileName := filepath.Join(o.OutputDir, name+".yaml")
	data, err := yaml.Marshal(build)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal Build YAML")
	}
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save build template file %s", fileName)
	}
	return nil
}

func (o *StepCreateBuildTemplateOptions) createSteps(languageName string, pipelineConfig *jenkinsfile.PipelineConfig, templateKind string, step *jenkinsfile.PipelineStep, containerName string, dir string) ([]corev1.Container, error) {

	steps := []corev1.Container{}

	if step.Container != "" {
		containerName = step.Container
	} else if step.Dir != "" {
		dir = step.Dir
	}
	if step.Command != "" {
		if containerName == "" {
			containerName = defaultContainerName
		}
		podTemplate := o.PodTemplates[containerName]
		if podTemplate == nil {
			o.MissingPodTemplates[containerName] = true
			podTemplate = o.PodTemplates[defaultContainerName]
		}
		containers := podTemplate.Spec.Containers
		if len(containers) == 0 {
			return steps, fmt.Errorf("No Containers for pod template %s", containerName)
		}
		c := containers[0]
		c.Command = []string{"/bin/sh"}
		c.Args = []string{"-c", step.Command}

		if strings.HasPrefix(dir, "./") {
			dir = "/workspace" + strings.TrimPrefix(dir, ".")
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join("/workspace", dir)
		}
		c.WorkingDir = dir

		// TODO use different image based on if its jx or not?
		c.Image = "jenkinsxio/jx:latest"
		steps = append(steps, c)
	}
	for _, s := range step.Steps {
		childSteps, err := o.createSteps(languageName, pipelineConfig, templateKind, s, containerName, dir)
		if err != nil {
			return steps, err
		}
		steps = append(steps, childSteps...)
	}
	return steps, nil
}
