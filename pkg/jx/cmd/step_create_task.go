package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

		# view the steps that would be created
		jx step create task --view

			`)
)

// StepCreateTaskOptions contains the command line flags
type StepCreateTaskOptions struct {
	StepOptions

	Pack           string
	Dir            string
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
	ViewSteps      bool
	Duration       time.Duration
	FromRepo       bool

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
	Tasks       []*pipelineapi.Task
	Resources   []*pipelineapi.PipelineResource
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
	cmd.Flags().StringVarP(&options.OutDir, "output", "o", "", "The directory to write the output to as YAML")
	cmd.Flags().StringVarP(&options.BuildPackURL, "url", "u", "", "The URL for the build pack Git repository")
	cmd.Flags().StringVarP(&options.BuildPackRef, "ref", "r", "", "The Git reference (branch,tag,sha) in the Git repository to use")
	cmd.Flags().StringVarP(&options.Pack, "pack", "p", "", "The build pack name. If none is specified its discovered from the source code")
	cmd.Flags().StringVarP(&options.Branch, "branch", "", "", "The git branch to trigger the build in. Defaults to the current local branch name")
	cmd.Flags().StringVarP(&options.PipelineKind, "kind", "k", "release", "The kind of pipeline to create such as: "+strings.Join(jenkinsfile.PipelineKinds, ", "))
	cmd.Flags().StringVarP(&options.Context, "context", "c", "", "The pipeline context if there are multiple separate pipelines for a given branch")
	cmd.Flags().StringVarP(&options.Trigger, "trigger", "t", string(pipelineapi.PipelineTriggerTypeManual), "The kind of pipeline trigger")
	cmd.Flags().StringVarP(&options.ServiceAccount, "service-account", "", "build-pipeline", "The Kubernetes ServiceAccount to use to run the pipeline")
	cmd.Flags().StringVarP(&options.DockerRegistry, "docker-registry", "", "", "The Docker Registry host name to use which is added as a prefix to docker images")
	cmd.Flags().StringVarP(&options.TargetPath, "target-path", "", "", "The target path appended to /workspace/${source} to clone the source code")
	cmd.Flags().StringVarP(&options.SourceName, "source", "", "source", "The name of the source repository")
	cmd.Flags().StringVarP(&options.CustomImage, "image", "", "", "Specify a custom image to use for the steps which overrides the image in the PodTemplates")
	cmd.Flags().StringVarP(&options.CloneGitURL, "clone-git-url", "", "", "Specify the git URL to clone to a temporary directory to get the source code")
	cmd.Flags().BoolVarP(&options.DeleteTempDir, "delete-temp-dir", "", false, "Deletes the temporary directory of cloned files if using the 'clone-git-url' option")
	cmd.Flags().BoolVarP(&options.NoApply, "no-apply", "", false, "Disables creating the Pipeline resources in the kubernetes cluster and just outputs the generated Task to the console or output file")
	cmd.Flags().BoolVarP(&options.ViewSteps, "view", "", false, "Just view the steps that would be created")
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

	if o.Dir == "" {
		o.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	// TODO: Best to separate things cleanly into 2 steps: creation of CRDs and
	// application of those CRDs to the cluster. Step 2 should be identical both
	// cases, so we'd just need a flag to switch the single function that is used
	// to generate stuff and then everything else would be identical.
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
	projectConfig, projectConfigFile, err := o.loadProjectConfig()
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

	pipelineFile := projectConfigFile
	pipelineConfig := projectConfig.PipelineConfig
	if name != "none" {
		pipelineFile := filepath.Join(packDir, jenkinsfile.PipelineConfigFileName)
		exists, err := util.FileExists(pipelineFile)
		if err != nil {
			return errors.Wrapf(err, "failed to find build pack pipeline YAML: %s", pipelineFile)
		}
		if !exists {
			return fmt.Errorf("no build pack for %s exists at directory %s", name, packDir)
		}
		jenkinsfileRunner := true
		pipelineConfig, err = jenkinsfile.LoadPipelineConfig(pipelineFile, resolver, jenkinsfileRunner, false)
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
	}
	err = o.generateTask(name, pipelineConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to generate Task for build pack pipeline YAML: %s", pipelineFile)
	}
	return err
}

func (o *StepCreateTaskOptions) loadProjectConfig() (*config.ProjectConfig, string, error) {
	if o.Context != "" {
		fileName := filepath.Join(o.Dir, fmt.Sprintf("jenkins-x-%s.yml", o.Context))
		exists, err := util.FileExists(fileName)
		if err != nil {
		  return nil, fileName, errors.Wrapf(err, "failed to check if file exists %s", fileName)
		}
		if exists {
			config, err := config.LoadProjectConfigFile(fileName)
			return config, fileName, err
		}
	}
	return config.LoadProjectConfig(o.Dir)
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

		// lets add a pre-step to setup the credentials
		if lifecycles.Setup == nil {
			lifecycles.Setup = &jenkinsfile.PipelineLifecycle{}
		}
		steps := []*jenkinsfile.PipelineStep{
			{
				Command: "jx step git credentials",
				Name:    "jx-git-credentials",
			},
		}
		lifecycles.Setup.Steps = append(steps, lifecycles.Setup.Steps...)

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
	err = o.setBuildValues()
	if err != nil {
		return err
	}

	// If there's an explicitly specified Pipeline in the lifecycle, use that.
	if lifecycles.Pipeline != nil {
		// TODO: Seeing weird behavior seemingly related to https://golang.org/doc/faq#nil_error
		// if err is reused, maybe we need to switch return types (perhaps upstream in build-pipeline)?
		if validateErr := lifecycles.Pipeline.Validate(); validateErr != nil {
			return errors.Wrapf(validateErr, "Validation failed for Pipeline")
		}
		err = o.setBuildValues()
		if err != nil {
			return err
		}
		// TODO: use org-name-branch for pipeline name? Create client now to get
		// namespace? Set namespace when applying rather than during generation?
		name := kpipelines.PipelineResourceName(o.gitInfo, o.Branch, o.Context)
		pipeline, tasks, _, err := kpipelines.GenerateCRDs(lifecycles.Pipeline, name, o.buildNumber, "will-be-replaced", "abcd", o.PodTemplates)
		if err != nil {
			return errors.Wrapf(err, "Generation failed for Pipeline")
		}

		if validateErr := pipeline.Spec.Validate(); validateErr != nil {
			return errors.Wrapf(validateErr, "Validation failed for generated Pipeline")
		}
		for _, task := range tasks {
			if validateErr := task.Spec.Validate(); validateErr != nil {
				return errors.Wrapf(validateErr, "Validation failed for generated Task: %s", task.Name)
			}

			var volumes []corev1.Volume
			for i, step := range task.Spec.Steps {
				volumes = o.modifyVolumes(&step, task.Spec.Volumes)
				o.modifyEnvVars(&step)
				task.Spec.Steps[i] = step
			}

			task.Spec.Volumes = volumes
		}

		if o.ViewSteps {
			return o.viewSteps(tasks...)
		}

		// TODO: where should this be created? In GenerateCRDs?
		var resources []*pipelineapi.PipelineResource
		resources = append(resources, o.generateSourceRepoResource(name), o.generateTempOrderingResource())

		err = o.applyPipeline(pipeline, tasks, resources, o.gitInfo, o.Branch)
		if err != nil {
			return errors.Wrapf(err, "failed to apply generated Pipeline")
		}

		folderName := o.OutDir
		if folderName != "" {
			err = o.writeOutput(folderName, o.Results.Pipeline, o.Results.Tasks, o.Results.PipelineRun, o.Results.Resources)
			if err != nil {
				return errors.Wrapf(err, "failed to write generated output to %s", folderName)
			}
		}
		return nil
	}

	// lets generate the pipeline using the build packs
	container := pipelineConfig.Agent.Container
	if o.CustomImage != "" {
		container = o.CustomImage
	}
	dir := o.getWorkspaceDir()

	steps := []corev1.Container{}
	volumes := []corev1.Volume{}
	for _, n := range lifecycles.All() {
		l := n.Lifecycle
		if l == nil {
			continue
		}
		for _, s := range l.Steps {
			ss, v, err := o.createSteps(languageName, pipelineConfig, templateKind, s, container, dir, n.Name)
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
			APIVersion: v1.PipelineAPIVersion,
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
	if task.Spec.Inputs == nil {
		task.Spec.Inputs = &pipelineapi.Inputs{}
	}
	sourceResourceName := o.SourceName
	task.Spec.Inputs.Resources = append(task.Spec.Inputs.Resources, pipelineapi.TaskResource{
		Name:       sourceResourceName,
		Type:       pipelineapi.PipelineResourceTypeGit,
		TargetPath: o.TargetPath,
	})

	if o.ViewSteps {
		return o.viewSteps(task)
	}
	err = o.applyTask(task, o.gitInfo, o.Branch)
	if err != nil {
		return errors.Wrapf(err, "failed to apply generated Pipeline")
	}

	folderName := o.OutDir
	if folderName != "" {
		err = o.writeOutput(folderName, o.Results.Pipeline, o.Results.Tasks, o.Results.PipelineRun, o.Results.Resources)
		if err != nil {
			return errors.Wrapf(err, "failed to write generated output to %s", folderName)
		}
	}

	return nil
}

func (o *StepCreateTaskOptions) generateSourceRepoResource(name string) *pipelineapi.PipelineResource {
	var resource *pipelineapi.PipelineResource
	if o.gitInfo != nil {
		gitURL := o.gitInfo.HttpsURL()
		if gitURL != "" {
			resource = &pipelineapi.PipelineResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.PipelineAPIVersion,
					Kind:       "PipelineResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: pipelineapi.PipelineResourceSpec{
					Type: pipelineapi.PipelineResourceTypeGit,
					Params: []pipelineapi.Param{
						{
							Name:  "revision",
							Value: o.Branch,
						},
						{
							Name:  "url",
							Value: gitURL,
						},
					},
				},
			}
		}
	}
	return resource
}

// TODO: This should not exist, but we need some way to enforce ordering of
// tasks and right now resources are the only way to do that.
func (o *StepCreateTaskOptions) generateTempOrderingResource() *pipelineapi.PipelineResource {
	return &pipelineapi.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.PipelineAPIVersion,
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "temp-ordering-resource",
		},
		Spec: pipelineapi.PipelineResourceSpec{
			Type: pipelineapi.PipelineResourceTypeImage,
			Params: []pipelineapi.Param{
				{
					Name:  "url",
					Value: "alpine", // Something smallish (lol)
				},
			},
		},
	}
}

func (o *StepCreateTaskOptions) setBuildValues() error {
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

	return nil
}

// TODO: Use the same YAML lib here as in buildpipeline/pipeline.go
// TODO: Use interface{} with a helper function to reduce code repetition?
// TODO: Take no arguments and use o.Results internally?
func (o *StepCreateTaskOptions) writeOutput(folder string, pipeline *pipelineapi.Pipeline, tasks []*pipelineapi.Task, pipelineRun *pipelineapi.PipelineRun, resources []*pipelineapi.PipelineResource) error {
	if err := os.Mkdir(folder, os.ModePerm); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	data, err := yaml.Marshal(pipeline)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal Pipeline YAML")
	}
	fileName := filepath.Join(folder, "pipeline.yml")
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save Pipeline file %s", fileName)
	}
	log.Infof("generated Pipeline at %s\n", util.ColorInfo(fileName))

	data, err = yaml.Marshal(pipelineRun)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal PipelineRun YAML")
	}
	fileName = filepath.Join(folder, "pipeline-run.yml")
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save PipelineRun file %s", fileName)
	}
	log.Infof("generated PipelineRun at %s\n", util.ColorInfo(fileName))

	for i, task := range tasks {
		data, err = yaml.Marshal(task)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal Task YAML")
		}
		fileName = filepath.Join(folder, fmt.Sprintf("task-%d.yml", i))
		err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save Task file %s", fileName)
		}
		log.Infof("generated Task at %s\n", util.ColorInfo(fileName))
	}

	for i, resource := range resources {
		data, err = yaml.Marshal(resource)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal PipelineResource YAML")
		}
		fileName = filepath.Join(folder, fmt.Sprintf("resource-%d.yml", i))
		err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save PipelineResource file %s", fileName)
		}
		log.Infof("generated PipelineResource at %s\n", util.ColorInfo(fileName))
	}

	return nil
}

func (o *StepCreateTaskOptions) applyTask(task *pipelineapi.Task, gitInfo *gits.GitRepository, branch string) error {
	organisation := gitInfo.Organisation
	name := gitInfo.Name
	resourceName := kube.ToValidName(organisation + "-" + name + "-" + branch)
	var pipelineResources []*pipelineapi.PipelineResource
	resource := o.generateSourceRepoResource(resourceName)
	if resource != nil {
		pipelineResources = append(pipelineResources, resource)
	}
	taskInputResources := []pipelineapi.PipelineTaskInputResource{}
	resources := []pipelineapi.PipelineDeclaredResource{}
	if resource != nil {
		resources = append(resources, pipelineapi.PipelineDeclaredResource{
			Name: resource.Name,
			Type: resource.Spec.Type,
		})
		taskInputResources = append(taskInputResources, pipelineapi.PipelineTaskInputResource{
			Name:     o.SourceName,
			Resource: resource.Name,
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

	pipeline := &pipelineapi.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: kpipelines.PipelineResourceName(gitInfo, branch, o.Context),
		},
		Spec: pipelineapi.PipelineSpec{
			Resources: resources,
			Tasks:     tasks,
		},
	}
	return o.applyPipeline(pipeline, []*pipelineapi.Task{task}, pipelineResources, gitInfo, branch)
}

// Given a Pipeline and its Tasks, applies the Tasks and Pipeline to the cluster
// and creates and applies a PipelineResource for their source repo and a PipelineRun
// to execute them. Handles o.NoApply internally.
// TODO: Probably needs to take PipelineResources as an input as well
func (o *StepCreateTaskOptions) applyPipeline(pipeline *pipelineapi.Pipeline, tasks []*pipelineapi.Task, resources []*pipelineapi.PipelineResource, gitInfo *gits.GitRepository, branch string) error {
	_, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	kpClient, _, err := o.KnativePipelineClient()
	if err != nil {
		return err
	}

	info := util.ColorInfo

	var resourceBindings []pipelineapi.PipelineResourceBinding
	for _, resource := range resources {
		resourceBindings = append(resourceBindings, pipelineapi.PipelineResourceBinding{
			Name: resource.Name,
			ResourceRef: pipelineapi.PipelineResourceRef{
				Name:       resource.Name,
				APIVersion: resource.APIVersion,
			},
		})

		if !o.NoApply {
			_, err := kpipelines.CreateOrUpdateSourceResource(kpClient, ns, resource)
			if err != nil {
				return errors.Wrapf(err, "failed to create/update PipelineResource %s in namespace %s", resource.Name, ns)
			}
			if resource.Spec.Type == pipelineapi.PipelineResourceTypeGit {
				gitURL := gitInfo.HttpCloneURL()
				log.Infof("upserted PipelineResource %s for the git repository %s and branch %s\n", info(resource.Name), info(gitURL), info(branch))
			} else {
				log.Infof("upserted PipelineResource %s\n", info(resource.Name))
			}
		}
	}

	for _, task := range tasks {
		task.ObjectMeta.Namespace = ns

		if !o.NoApply {
			_, err = kpipelines.CreateOrUpdateTask(kpClient, ns, task)
			if err != nil {
				return errors.Wrapf(err, "failed to create/update the task %s in namespace %s", task.Name, ns)
			}
			log.Infof("upserted Task %s\n", info(task.Name))
		}
	}

	pipeline.ObjectMeta.Namespace = ns
	if pipeline.APIVersion == "" {
		pipeline.APIVersion = v1.PipelineAPIVersion
	}
	if pipeline.Kind == "" {
		pipeline.Kind = "Pipeline"
	}
	if !o.NoApply {
		// TODO: Result is missing some fields that the original has, such as APIVersion and Kind. Why?
		pipeline, err = kpipelines.CreateOrUpdatePipeline(kpClient, ns, pipeline, o.labels)
		if err != nil {
			return errors.Wrapf(err, "failed to create/update the Pipeline in namespace %s", ns)
		}
		log.Infof("upserted Pipeline %s\n", info(pipeline.Name))
	}

	run := &pipelineapi.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.PipelineAPIVersion,
			Kind:       "PipelineRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   pipeline.Name,
			Labels: util.MergeMaps(o.labels),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1.PipelineAPIVersion,
					Kind:       "Pipeline",
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

	if !o.NoApply {
		_, err = kpipelines.CreatePipelineRun(kpClient, ns, pipeline, run, o.Duration)
		if err != nil {
			return errors.Wrapf(err, "failed to create the PipelineRun in namespace %s", ns)
		}
		log.Infof("created PipelineRun %s\n", info(run.Name))
	}

	o.Results.Tasks = tasks
	o.Results.Pipeline = pipeline
	o.Results.Resources = resources
	o.Results.PipelineRun = run
	return nil
}

func (o *StepCreateTaskOptions) createSteps(languageName string, pipelineConfig *jenkinsfile.PipelineConfig, templateKind string, step *jenkinsfile.PipelineStep, containerName string, dir string, prefixPath string) ([]corev1.Container, []corev1.Volume, error) {
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

		prefix := prefixPath
		if prefix != "" {
			prefix += "-"
		}
		stepName := step.Name
		if stepName == "" {
			stepName = "step" + strconv.Itoa(1+o.stepCounter)
		}
		c.Name = prefix + stepName

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
		c.TTY = false

		steps = append(steps, c)
	}
	for _, s := range step.Steps {
		// TODO add child prefix?
		childPrefixPath := prefixPath
		childSteps, v, err := o.createSteps(languageName, pipelineConfig, templateKind, s, containerName, dir, childPrefixPath)
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
		if name != "JENKINS_URL" {
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
			envVars = append(envVars, corev1.PipelineStructureEnvVar{
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

func (o *StepCreateTaskOptions) viewSteps(tasks ...*pipelineapi.Task) error {
	table := o.createTable()
	showTaskName := len(tasks) > 1
	if showTaskName {
		table.AddRow("TASK", "NAME", "COMMAND", "IMAGE")
	} else {
		table.AddRow("NAME", "COMMAND", "IMAGE")
	}
	for _, task := range tasks {
		for _, step := range task.Spec.Steps {
			command := append([]string{}, step.Command...)
			command = append(command, step.Args...)
			commands := strings.Join(command, " ")
			if showTaskName {
				table.AddRow(task.Name, step.Name, commands, step.Image)
			} else {
				table.AddRow(step.Name, commands, step.Image)
			}
		}
	}
	table.Render()
	return nil
}

// ObjectReferences creates a list of object references created
func (r *StepCreateTaskResults) ObjectReferences() []kube.ObjectReference {
	resources := []kube.ObjectReference{}
	for _, task := range r.Tasks {
		resources = append(resources, kube.CreateObjectReference(task.TypeMeta, task.ObjectMeta))
	}
	if r.Pipeline != nil {
		resources = append(resources, kube.CreateObjectReference(r.Pipeline.TypeMeta, r.Pipeline.ObjectMeta))
	}
	if r.PipelineRun != nil {
		resources = append(resources, kube.CreateObjectReference(r.PipelineRun.TypeMeta, r.PipelineRun.ObjectMeta))
	}
	return resources
}
