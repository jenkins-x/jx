package cmd

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jenkinsfile/gitresolver"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kpipelines"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	pipelineapi "github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	createTaskLong = templates.LongDesc(`
		Creates a Knative Pipeline Task for a project
`)

	createTaskExample = templates.Examples(`
		# create a Knative Pipeline Task and render to the console
		jx step create task

		# create a Knative Pipeline Task
		jx step create task -o mytask.yaml

			`)
)

// StepCreateTaskOptions contains the command line flags
type StepCreateTaskOptions struct {
	StepOptions

	Pack         string
	Dir          string
	OutputFile   string
	BuildPackURL string
	BuildPackRef string
	PipelineKind string
	Context      string
	Apply        bool
	Trigger      string
	TargetPath   string
	Duration     time.Duration

	PodTemplates        map[string]*corev1.Pod
	MissingPodTemplates map[string]bool

	stepCounter int
}

// NewCmdStepCreateTask Creates a new Command object
func NewCmdStepCreateTask(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepCreateTaskOptions{
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
		Use:     "task",
		Short:   "Creates a Knative Pipeline Task for the current folder or given build pack",
		Long:    createTaskLong,
		Example: createTaskExample,
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
	cmd.Flags().StringVarP(&options.OutputFile, "output", "o", "", "The output file to write the output to as YAML")
	cmd.Flags().StringVarP(&options.BuildPackURL, "url", "u", "", "The URL for the build pack Git repository")
	cmd.Flags().StringVarP(&options.BuildPackRef, "ref", "r", "", "The Git reference (branch,tag,sha) in the Git repository to use")
	cmd.Flags().StringVarP(&options.Pack, "pack", "p", "", "The build pack name. If none is specified its discovered from the source code")
	cmd.Flags().StringVarP(&options.PipelineKind, "kind", "k", "release", "The kind of pipeline to create such as: "+strings.Join(jenkinsfile.PipelineKinds, ", "))
	cmd.Flags().StringVarP(&options.Context, "context", "c", "", "The pipeline context if there are multiple separate pipelines for a given branch")
	cmd.Flags().StringVarP(&options.Trigger, "trigger", "t", string(pipelineapi.PipelineTriggerTypeManual), "The kind of pipeline trigger")
	cmd.Flags().StringVarP(&options.ServiceAccount, "service-account", "", "build-pipeline", "The Kubernetes ServiceAccount to use to run the pipeline")
	cmd.Flags().StringVarP(&options.TargetPath, "target-path", "", "", "The target path to expose the source code")
	cmd.Flags().BoolVarP(&options.Apply, "apply", "a", false, "If enabled lets apply the generated")
	cmd.Flags().DurationVarP(&options.Duration, "duration", "", time.Second*30, "Retry duration when trying to create a PipelineRun")
	return cmd
}

// Run implements this command
func (o *StepCreateTaskOptions) Run() error {
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	if o.BuildPackURL == "" || o.BuildPackRef == "" {
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
	if o.PipelineKind == "" {
		return util.MissingOption("kind")
	}
	if o.Dir == "" {
		o.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	projectConfig, projectConfigFile, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load project config in dir %s", o.Dir)
	}
	if o.Pack == "" {
		o.Pack = projectConfig.BuildPack
	}
	if o.Pack == "" {
		o.Pack, err = o.discoverBuildPack(o.Dir, projectConfig)
	}

	if o.Pack == "" {
		return util.MissingOption("pack")
	}

	err = o.loadPodTemplates()
	if err != nil {
		return err
	}
	o.MissingPodTemplates = map[string]bool{}

	packsDir, err := gitresolver.InitBuildPack(o.Git(), o.BuildPackURL, o.BuildPackRef)
	if err != nil {
		return err
	}

	resolver, err := gitresolver.CreateResolver(packsDir, o.Git())
	if err != nil {
		return err
	}

	name := o.Pack
	packDir := filepath.Join(packsDir, name)

	pipelineFile := filepath.Join(packDir, jenkinsfile.PipelineConfigFileName)
	exists, err := util.FileExists(pipelineFile)
	if err != nil {
		return errors.Wrapf(err, "failed to find build pack pipeline YAML: %s", pipelineFile)
	}
	if !exists {
		return fmt.Errorf("no build pack for %s exists at directory %s", name, packDir)
	}
	jenkinsfileRunner := true
	pipelineConfig, err := jenkinsfile.LoadPipelineConfig(pipelineFile, resolver, jenkinsfileRunner)
	if err != nil {
		return errors.Wrapf(err, "failed to load build pack pipeline YAML: %s", pipelineFile)
	}
	localPipelineConfig := projectConfig.PipelineConfig
	if localPipelineConfig != nil {
		err = localPipelineConfig.ExtendPipeline(pipelineConfig, jenkinsfileRunner)
		if err != nil {
			return errors.Wrapf(err, "failed to override PipelineConfig using configuration in file %s", projectConfigFile)
		}
		pipelineConfig = localPipelineConfig
	}
	err = o.generateTask(name, pipelineConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to generate Task for build pack pipeline YAML: %s", pipelineFile)
	}
	return err
}

func (o *StepCreateTaskOptions) loadPodTemplates() error {
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

func (o *StepCreateTaskOptions) generateTask(name string, pipelineConfig *jenkinsfile.PipelineConfig) error {
	var lifecycles *jenkinsfile.PipelineLifecycles
	kind := o.PipelineKind
	pipelines := pipelineConfig.Pipelines
	switch kind {
	case jenkinsfile.PipelineKindRelease:
		lifecycles = pipelines.Release
	case jenkinsfile.PipelineKindPullRequest:
		lifecycles = pipelines.PullRequest
	case jenkinsfile.PipelineKindFeature:
		lifecycles = pipelines.Feature
	default:
		return fmt.Errorf("Unknown pipeline kind %s. Supported values are %s", kind, strings.Join(jenkinsfile.PipelineKinds, ", "))
	}
	return o.generatePipeline(name, pipelineConfig, lifecycles, kind)
}

func (o *StepCreateTaskOptions) generatePipeline(languageName string, pipelineConfig *jenkinsfile.PipelineConfig, lifecycles *jenkinsfile.PipelineLifecycles, templateKind string) error {
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

	gitInfo, err := o.FindGitInfo(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to find git information from dir %s", o.Dir)
	}

	branch, err := o.Git().Branch(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to find git branch from dir %s", o.Dir)
	}

	name := kpipelines.PipelineResourceName(gitInfo, branch, o.Context)
	task := &pipelineapi.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "pipeline.knative.dev/v1alpha1",
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: pipelineapi.TaskSpec{
			Steps: steps,
		},
	}
	fileName := o.OutputFile
	if o.Apply {
		err = o.applyTask(task, gitInfo, branch)
		if fileName == "" {
			return err
		}
		err2 := o.writeTask(fileName, task)
		return util.CombineErrors(err, err2)
	}

	if fileName == "" {
		data, err := yaml.Marshal(task)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal Task YAML")
		}
		log.Infof("%s\n", string(data))
		return nil
	}
	return o.writeTask(fileName, task)
}

func (o *StepCreateTaskOptions) writeTask(fileName string, task *pipelineapi.Task) error {
	data, err := yaml.Marshal(task)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal Task YAML")
	}
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save Task file %s", fileName)
	}
	log.Infof("generated Task at %s\n", util.ColorInfo(fileName))
	return nil
}

func (o *StepCreateTaskOptions) applyTask(task *pipelineapi.Task, gitInfo *gits.GitRepository, branch string) error {
	_, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	kpClient, _, err := o.KnativePipelineClient()
	if err != nil {
		return err
	}

	resource, err := kpipelines.CreateOrUpdateSourceResource(kpClient, ns, gitInfo, branch)
	if err != nil {
		return errors.Wrapf(err, "failed to create/update the PipelineResource in namespace %s", ns)
	}

	gitURL := gitInfo.HttpCloneURL()
	info := util.ColorInfo
	log.Infof("upserted PipelineResource %s for the git repository %s and branch %s\n", info(resource.Name), info(gitURL), info(branch))

	if task.Spec.Inputs == nil {
		task.Spec.Inputs = &pipelineapi.Inputs{}
	}
	sourceResourceName := "source"
	task.Spec.Inputs.Resources = append(task.Spec.Inputs.Resources, pipelineapi.TaskResource{
		Name:       sourceResourceName,
		Type:       pipelineapi.PipelineResourceTypeGit,
		TargetPath: o.TargetPath,
	})

	_, err = kpipelines.CreateOrUpdateTask(kpClient, ns, task)
	if err != nil {
		return errors.Wrapf(err, "failed to create/update the task %s in namespace %s", task.Name, ns)
	}
	log.Infof("upserted Task %s\n", info(task.Name))

	taskInputResources := []pipelineapi.PipelineTaskInputResource{}
	resources := []pipelineapi.PipelineDeclaredResource{}
	resourceBindings := []pipelineapi.PipelineResourceBinding{}

	if resource != nil {
		resources = append(resources, pipelineapi.PipelineDeclaredResource{
			Name: resource.Name,
			Type: resource.Spec.Type,
		})
		taskInputResources = append(taskInputResources, pipelineapi.PipelineTaskInputResource{
			Name:     sourceResourceName,
			Resource: resource.Name,
		})
		resourceBindings = append(resourceBindings, pipelineapi.PipelineResourceBinding{
			Name: resource.Name,
			ResourceRef: pipelineapi.PipelineResourceRef{
				Name:       resource.Name,
				APIVersion: resource.APIVersion,
			},
		})
	}
	tasks := []pipelineapi.PipelineTask{
		{
			Name: "build",
			Resources: &pipelineapi.PipelineTaskResources{
				Inputs: taskInputResources,
			},
			TaskRef: pipelineapi.TaskRef{
				Name:       task.Name,
				Kind:       pipelineapi.NamespacedTaskKind,
				APIVersion: task.APIVersion,
			},
		},
	}

	// lets lazily create the Pipeline
	pipeline, err := kpipelines.CreateOrUpdatePipeline(kpClient, ns, gitInfo, branch, o.Context, resources, tasks)
	if err != nil {
		return errors.Wrapf(err, "failed to create/update the Pipeline in namespace %s", ns)
	}
	log.Infof("upserted Pipeline %s\n", info(pipeline.Name))

	run := &pipelineapi.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "pipeline.knative.dev/v1alpha1",
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pipeline.Name,
		},
		Spec: pipelineapi.PipelineRunSpec{
			ServiceAccount: o.ServiceAccount,
			Trigger: pipelineapi.PipelineTrigger{
				Type: pipelineapi.PipelineTriggerType(o.Trigger),
			},
			PipelineRef: pipelineapi.PipelineRef{
				Name:       pipeline.Name,
				APIVersion: pipeline.APIVersion,
			},
			Resources: resourceBindings,
		},
	}

	_, err = kpipelines.CreatePipelineRun(kpClient, ns, pipeline, run, o.Duration)
	if err != nil {
		return errors.Wrapf(err, "failed to create the PipelineRun namespace %s", ns)
	}
	log.Infof("created PipelineRun %s\n", info(run.Name))
	return nil
}

func (o *StepCreateTaskOptions) createSteps(languageName string, pipelineConfig *jenkinsfile.PipelineConfig, templateKind string, step *jenkinsfile.PipelineStep, containerName string, dir string) ([]corev1.Container, error) {

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
		o.stepCounter++
		c.Name = "step" + strconv.Itoa(1+o.stepCounter)

		o.removeUnnecessaryVolumes(&c)
		o.removeUnnecessaryEnvVars(&c)

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

func (o *StepCreateTaskOptions) discoverBuildPack(dir string, projectConfig *config.ProjectConfig) (string, error) {
	args := &InvokeDraftPack{
		Dir:             o.Dir,
		CustomDraftPack: o.Pack,
		ProjectConfig:   projectConfig,
		DisableAddFiles: true,
	}
	pack, err := o.invokeDraftPack(args)
	if err != nil {
		return pack, errors.Wrapf(err, "failed to discover task pack in dir %s", o.Dir)
	}
	return pack, nil
}

func (o *StepCreateTaskOptions) removeUnnecessaryVolumes(container *corev1.Container) {
	// for now let remove them all?
	container.VolumeMounts = nil
}

func (o *StepCreateTaskOptions) removeUnnecessaryEnvVars(container *corev1.Container) {
	envVars := []corev1.EnvVar{}
	for _, e := range container.Env {
		name := e.Name
		if strings.HasPrefix(name, "GIT_") || strings.HasPrefix(name, "DOCKER_") || strings.HasPrefix(name, "XDG_") {
			continue
		}
		envVars = append(envVars, e)
	}
	container.Env = envVars
}
