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
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	createTaskLong = templates.LongDesc(`
		Creates a Knative Pipeline Run for a project
`)

	createTaskExample = templates.Examples(`
		# create a Knative Pipeline Run and render to the console
		jx step create task

		# create a Knative Pipeline Task
		jx step create task -o mytask.yaml

			`)
)

// StepCreateTaskOptions contains the command line flags
type StepCreateTaskOptions struct {
	StepOptions

	Pack           string
	Dir            string
	OutputFile     string
	BuildPackURL   string
	BuildPackRef   string
	PipelineKind   string
	Context        string
	NoApply        bool
	Trigger        string
	TargetPath     string
	SourceName     string
	CustomImage    string
	DockerRegistry string
	CloneGitURL    string
	Branch         string
	DeleteTempDir  bool
	Duration       time.Duration

	PodTemplates        map[string]*corev1.Pod
	MissingPodTemplates map[string]bool

	stepCounter int
	gitInfo     *gits.GitRepository
	buildNumber string
	labels      map[string]string
	Results     StepCreateTaskResults
}

// StepCreateTaskResults stores the generated results
type StepCreateTaskResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
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
		Short:   "Creates a Knative Pipeline Run for the current folder or given build pack",
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
	cmd.Flags().StringVarP(&options.Branch, "branch", "", "", "The git branch to trigger the build in. Defaults to the current local branch name")
	cmd.Flags().StringVarP(&options.PipelineKind, "kind", "k", "release", "The kind of pipeline to create such as: "+strings.Join(jenkinsfile.PipelineKinds, ", "))
	cmd.Flags().StringVarP(&options.Context, "context", "c", "", "The pipeline context if there are multiple separate pipelines for a given branch")
	cmd.Flags().StringVarP(&options.Trigger, "trigger", "t", string(pipelineapi.PipelineTriggerTypeManual), "The kind of pipeline trigger")
	cmd.Flags().StringVarP(&options.ServiceAccount, "service-account", "", "pipeline", "The Kubernetes ServiceAccount to use to run the pipeline")
	cmd.Flags().StringVarP(&options.DockerRegistry, "docker-registry", "", "", "The Docker Registry host name to use which is added as a prefix to docker images")
	cmd.Flags().StringVarP(&options.TargetPath, "target-path", "", "", "The target path appended to /workspace/${source} to clone the source code")
	cmd.Flags().StringVarP(&options.SourceName, "source", "", "source", "The name of the source repository")
	cmd.Flags().StringVarP(&options.CustomImage, "image", "", "", "Specify a custom image to use for the steps which overrides the image in the PodTemplates")
	cmd.Flags().StringVarP(&options.CloneGitURL, "clone-git-url", "", "", "Specify the git URL to clone to a temporary directory to get the source code")
	cmd.Flags().BoolVarP(&options.DeleteTempDir, "delete-temp-dir", "", false, "Deletes the temporary directory of cloned files if using the 'clone-git-url' option")
	cmd.Flags().BoolVarP(&options.NoApply, "no-apply", "", false, "Disables creating the Pipeline resources in the kubernetes cluster and just outputs the generated Task to the console or output file")
	cmd.Flags().DurationVarP(&options.Duration, "duration", "", time.Second*30, "Retry duration when trying to create a PipelineRun")
	return cmd
}

// Run implements this command
func (o *StepCreateTaskOptions) Run() error {
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}

	if o.CloneGitURL != "" {
		err = o.cloneGitRepositoryToTempDir(o.CloneGitURL)
		if err != nil {
			return err
		}
		if o.DeleteTempDir {
			defer o.deleteTempDir()
		}
	}

	if o.DockerRegistry == "" {
		data, err := kube.GetConfigMapData(kubeClient, kube.ConfigMapJenkinsDockerRegistry, ns)
		if err != nil {
			return fmt.Errorf("Could not find ConfigMap %s in namespace %s: %s", kube.ConfigMapJenkinsDockerRegistry, ns, err)
		}
		o.DockerRegistry = data["docker.registry"]
		if o.DockerRegistry == "" {
			return util.MissingOption("docker-registry")
		}
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

	err = o.loadPodTemplates(kubeClient, ns)
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
	pipelineConfig, err := jenkinsfile.LoadPipelineConfig(pipelineFile, resolver, jenkinsfileRunner, false)
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

func (o *StepCreateTaskOptions) loadPodTemplates(kubeClient kubernetes.Interface, ns string) error {
	o.PodTemplates = map[string]*corev1.Pod{}

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

	var err error
	o.gitInfo, err = o.FindGitInfo(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to find git information from dir %s", o.Dir)
	}

	if o.Branch == "" {
		o.Branch, err = o.Git().Branch(o.Dir)
		if err != nil {
			return errors.Wrapf(err, "failed to find git branch from dir %s", o.Dir)
		}
	}

	// TODO generate build number properly!
	o.buildNumber = "1"

	labels := map[string]string{}
	if o.gitInfo != nil {
		labels["owner"] = o.gitInfo.Organisation
		labels["repo"] = o.gitInfo.Name
	}
	labels["branch"] = o.Branch
	o.labels = labels

	container := pipelineConfig.Agent.Container
	if o.CustomImage != "" {
		container = o.CustomImage
	}
	dir := o.getWorkspaceDir()

	steps := []corev1.Container{}
	volumes := []corev1.Volume{}
	for _, l := range lifecycles.All() {
		if l == nil {
			continue
		}
		for _, s := range l.Steps {
			ss, v, err := o.createSteps(languageName, pipelineConfig, templateKind, s, container, dir)
			if err != nil {
				return err
			}
			steps = append(steps, ss...)
			volumes = kube.CombineVolumes(volumes, v...)
		}
	}

	name := kpipelines.PipelineResourceName(o.gitInfo, o.Branch, o.Context)
	task := &pipelineapi.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kpipelines.PipelineAPIVersion,
			Kind:       "Task",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: o.labels,
		},
		Spec: pipelineapi.TaskSpec{
			Steps:   steps,
			Volumes: volumes,
		},
	}
	fileName := o.OutputFile
	if !o.NoApply {
		err = o.applyTask(task, o.gitInfo, o.Branch)
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
	sourceResourceName := o.SourceName
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
	pipeline, err := kpipelines.CreateOrUpdatePipeline(kpClient, ns, gitInfo, branch, o.Context, resources, tasks, o.labels)
	if err != nil {
		return errors.Wrapf(err, "failed to create/update the Pipeline in namespace %s", ns)
	}
	log.Infof("upserted Pipeline %s\n", info(pipeline.Name))

	if pipeline.APIVersion == "" {
		pipeline.APIVersion = kpipelines.PipelineAPIVersion
	}
	if pipeline.Kind == "" {
		pipeline.Kind = "Pipeline"
	}
	run := &pipelineapi.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "pipeline.knative.dev/v1alpha1",
			Kind:       "PipelineRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   pipeline.Name,
			Labels: util.MergeMaps(o.labels),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: pipeline.APIVersion,
					Kind:       pipeline.Kind,
					Name:       pipeline.Name,
					UID:        pipeline.UID,
				},
			},
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

	o.Results.Task = task
	o.Results.Pipeline = pipeline
	o.Results.PipelineRun = run
	return nil
}

func (o *StepCreateTaskOptions) createSteps(languageName string, pipelineConfig *jenkinsfile.PipelineConfig, templateKind string, step *jenkinsfile.PipelineStep, containerName string, dir string) ([]corev1.Container, []corev1.Volume, error) {

	volumes := []corev1.Volume{}
	steps := []corev1.Container{}

	if step.Container != "" {
		containerName = step.Container
	} else if step.Dir != "" {
		dir = step.Dir
	}

	gitInfo := o.gitInfo
	if gitInfo != nil {
		dir = strings.Replace(dir, "REPLACE_ME_APP_NAME", gitInfo.Name, -1)
		dir = strings.Replace(dir, "REPLACE_ME_ORG_NAME", gitInfo.Organisation, -1)
	} else {
		log.Warnf("No GitInfo available!\n")
	}

	if step.Command != "" {
		if containerName == "" {
			containerName = defaultContainerName
			log.Warnf("No 'agent.container' specified in the pipeline configuration so defaulting to use: %s\n", containerName)
		}
		podTemplate := o.PodTemplates[containerName]
		if podTemplate == nil {
			log.Warnf("Could not find a pod template for containerName %s\n", containerName)
			o.MissingPodTemplates[containerName] = true
			podTemplate = o.PodTemplates[defaultContainerName]
		}
		containers := podTemplate.Spec.Containers
		if len(containers) == 0 {
			return steps, volumes, fmt.Errorf("No Containers for pod template %s", containerName)
		}
		volumes = podTemplate.Spec.Volumes
		c := containers[0]
		o.stepCounter++
		c.Name = "step" + strconv.Itoa(1+o.stepCounter)

		volumes = o.modifyVolumes(&c, volumes)
		o.modifyEnvVars(&c)

		c.Command = []string{"/bin/sh"}
		if o.CustomImage != "" {
			c.Image = o.CustomImage
		}

		// lets remove any escaped "\$" stuff in the pipeline library
		commandText := strings.Replace(step.Command, "\\$", "$", -1)
		c.Args = []string{"-c", commandText}

		workspaceDir := o.getWorkspaceDir()
		if strings.HasPrefix(dir, "./") {
			dir = workspaceDir + strings.TrimPrefix(dir, ".")
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(workspaceDir, dir)
		}
		c.WorkingDir = dir
		c.Stdin = false

		steps = append(steps, c)
	}
	for _, s := range step.Steps {
		childSteps, v, err := o.createSteps(languageName, pipelineConfig, templateKind, s, containerName, dir)
		if err != nil {
			return steps, v, err
		}
		steps = append(steps, childSteps...)
		volumes = kube.CombineVolumes(volumes, v...)
	}
	return steps, volumes, nil
}

func (o *StepCreateTaskOptions) getWorkspaceDir() string {
	return filepath.Join("/workspace", o.SourceName)
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

func (o *StepCreateTaskOptions) modifyEnvVars(container *corev1.Container) {
	envVars := []corev1.EnvVar{}
	for _, e := range container.Env {
		name := e.Name
		if name != "JENKINS_URL" && !strings.HasPrefix(name, "XDG_") {
			envVars = append(envVars, e)
		}
	}
	if kube.GetSliceEnvVar(envVars, "DOCKER_REGISTRY") == nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "DOCKER_REGISTRY",
			Value: o.DockerRegistry,
		})
	}
	/*	if kube.GetSliceEnvVar(envVars, "BUILD_NUMBER") == nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BUILD_NUMBER",
				Value: o.buildNumber,
			})
		}
	*/
	if o.PipelineKind != "" && kube.GetSliceEnvVar(envVars, "PIPELINE_KIND") == nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "PIPELINE_KIND",
			Value: o.PipelineKind,
		})
	}
	if o.Context != "" && kube.GetSliceEnvVar(envVars, "PIPELINE_CONTEXT") == nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "PIPELINE_CONTEXT",
			Value: o.Context,
		})
	}
	gitInfo := o.gitInfo
	branch := o.Branch
	if gitInfo != nil {
		u := gitInfo.CloneURL
		if u != "" && kube.GetSliceEnvVar(envVars, "SOURCE_URL") == nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "SOURCE_URL",
				Value: u,
			})
		}
		repo := gitInfo.Name
		owner := gitInfo.Organisation
		if owner != "" && kube.GetSliceEnvVar(envVars, "REPO_OWNER") == nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "REPO_OWNER",
				Value: owner,
			})
		}
		if repo != "" && kube.GetSliceEnvVar(envVars, "REPO_NAME") == nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "REPO_NAME",
				Value: repo,
			})
		}
		if owner != "" && repo != "" && branch != "" {
			jobName := fmt.Sprintf("%s/%s/%s", owner, repo, branch)
			if kube.GetSliceEnvVar(envVars, "JOB_NAME") == nil {
				envVars = append(envVars, corev1.EnvVar{
					Name:  "JOB_NAME",
					Value: jobName,
				})
			}
		}
	}
	if branch != "" {
		if kube.GetSliceEnvVar(envVars, "BRANCH_NAME") == nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BRANCH_NAME",
				Value: branch,
			})
		}
	}
	if kube.GetSliceEnvVar(envVars, "JX_BATCH_MODE") == nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "JX_BATCH_MODE",
			Value: "true",
		})
	}
	container.Env = envVars
}

func (o *StepCreateTaskOptions) modifyVolumes(container *corev1.Container, volumes []corev1.Volume) []corev1.Volume {
	answer := volumes
	podInfoName := "podinfo"
	volume := corev1.Volume{
		Name: podInfoName,
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path: "labels",
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.labels",
						},
					},
				},
			},
		},
	}
	if !kube.ContainsVolume(volumes, volume) {
		answer = append(answer, volume)
	}
	volumeMount := corev1.VolumeMount{
		Name:      podInfoName,
		MountPath: "/etc/podinfo",
		ReadOnly:  true,
	}
	if !kube.ContainsVolumeMount(container.VolumeMounts, volumeMount) {
		container.VolumeMounts = append(container.VolumeMounts, volumeMount)
	}
	return answer
}

func (o *StepCreateTaskOptions) cloneGitRepositoryToTempDir(gitURL string) error {
	var err error
	o.Dir, err = ioutil.TempDir("", "git")
	if err != nil {
		return err
	}
	log.Infof("cloning repository %s to temp dir %s\n", gitURL, o.Dir)
	err = o.Git().ShallowCloneBranch(gitURL, o.Branch, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to clone repository %s to directory %s", gitURL, o.Dir)
	}
	return nil

}

func (o *StepCreateTaskOptions) deleteTempDir() {
	log.Infof("removing the temp directory %s\n", o.Dir)
	err := util.DeleteDirContents(o.Dir)
	if err != nil {
		log.Warnf("failed to delete dir %s: %s\n", o.Dir, err.Error())
	}
}

// ObjectReferences creates a list of object references created
func (r *StepCreateTaskResults) ObjectReferences() []kube.ObjectReference {
	resources := []kube.ObjectReference{}
	if r.Task != nil {
		resources = append(resources, kube.CreateObjectReference(r.Task.TypeMeta, r.Task.ObjectMeta))
	}
	if r.Pipeline != nil {
		resources = append(resources, kube.CreateObjectReference(r.Pipeline.TypeMeta, r.Pipeline.ObjectMeta))
	}
	if r.PipelineRun != nil {
		resources = append(resources, kube.CreateObjectReference(r.PipelineRun.TypeMeta, r.PipelineRun.ObjectMeta))
	}
	return resources
}
