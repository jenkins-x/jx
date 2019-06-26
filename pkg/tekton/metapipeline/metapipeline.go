package metapipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	appExtensionStageName = "app-extension"
	// createEffectivePipelineStepName is the meta pipeline step name for the generation of the effective jenkins-x pipeline config
	createEffectivePipelineStepName = "create-effective-pipeline"
	// createTektonCRDsStepName is the meta pipeline step name for the Tekton CRD creation
	createTektonCRDsStepName = "create-tekton-crds"

	tektonBaseDir = "/workspace"
)

// CRDCreationParameters are the parameters needed to create the Tekton CRDs
type CRDCreationParameters struct {
	Namespace      string
	Context        string
	PipelineName   string
	PipelineKind   string
	BuildNumber    string
	GitInfo        *gits.GitRepository
	Branch         string
	PullRef        string
	SourceDir      string
	PodTemplates   map[string]*corev1.Pod
	Trigger        string
	ServiceAccount string
	Labels         []string
	EnvVars        []string
	DefaultImage   string
	Apps           []jenkinsv1.App
}

// CreateMetaPipelineCRDs creates the Tekton CRDs needed to execute the meta pipeline.
// The meta pipeline is responsible to checkout the source repository at the right revision, allows Jenkins-X Apps
// to modify the pipeline (via modifying the configuration on the file system) and finally triggering the actual
// pipeline build.
// An error is returned in case the creation of the Tekton CRDs fails.
func CreateMetaPipelineCRDs(params CRDCreationParameters) (*tekton.CRDWrapper, error) {
	parsedPipeline, err := createPipeline(params)
	if err != nil {
		return nil, err
	}

	labels, err := buildLabels(params)
	if err != nil {
		return nil, err
	}
	pipeline, tasks, structure, err := parsedPipeline.GenerateCRDs(params.PipelineName, params.BuildNumber, params.Namespace, params.PodTemplates, nil, params.SourceDir, labels, params.DefaultImage)
	if err != nil {
		return nil, err
	}

	resources := []*pipelineapi.PipelineResource{tekton.GenerateSourceRepoResource(params.PipelineName, params.GitInfo, "")}
	run := tekton.CreatePipelineRun(resources, pipeline.Name, pipeline.APIVersion, labels, params.Trigger, params.ServiceAccount, nil, nil)

	tektonCRDs, err := tekton.NewCRDWrapper(pipeline, tasks, resources, structure, run)
	if err != nil {
		return nil, err
	}

	return tektonCRDs, nil
}

// GetExtendingApps returns the list of apps which are installed in the cluster registered for extending the pipeline.
// An app registers its interest in extending the pipeline by having the 'pipeline-extension' label set.
func GetExtendingApps(jxClient versioned.Interface, namespace string) ([]jenkinsv1.App, error) {
	listOptions := metav1.ListOptions{}
	listOptions.LabelSelector = fmt.Sprintf(apps.AppTypeLabel+" in (%s)", apps.PipelineExtension)
	appsList, err := jxClient.JenkinsV1().Apps(namespace).List(listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving pipeline contributor apps")
	}
	return appsList.Items, nil
}

// createPipeline builds the parsed/typed pipeline which servers as input for the Tekton CRD creation.
func createPipeline(params CRDCreationParameters) (*syntax.ParsedPipeline, error) {
	steps, err := buildSteps(params)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create app extending pipeline steps")
	}

	stage := syntax.Stage{
		Name:  appExtensionStageName,
		Steps: steps,
		Agent: &syntax.Agent{
			Image: determineDefaultStepImage(params.DefaultImage),
		},
	}

	parsedPipeline := &syntax.ParsedPipeline{
		Stages: []syntax.Stage{stage},
	}

	env := buildEnvParams(params)
	parsedPipeline.AddContainerEnvVarsToPipeline(env)

	return parsedPipeline, nil
}

// buildSteps builds a step (container) for each of the apps specified.
func buildSteps(params CRDCreationParameters) ([]syntax.Step, error) {
	var steps []syntax.Step

	step := stepEffectivePipeline(params)
	steps = append(steps, step)

	log.Logger().Debugf("creating pipeline steps for extending apps")
	for _, app := range params.Apps {
		if app.Spec.PipelineExtension == nil {
			log.Logger().Warnf("Skipping app %s in meta pipeline. It contains label %s with value %s, but does not contain PipelineExtension fields.", app.Name, apps.AppTypeLabel, apps.PipelineExtension)
			continue
		}

		extension := app.Spec.PipelineExtension
		step := syntax.Step{
			Name:      extension.Name,
			Image:     extension.Image,
			Command:   extension.Command,
			Arguments: extension.Args,
		}

		log.Logger().Debugf("App %s contributes with step %s", app.Name, util.PrettyPrint(step))
		steps = append(steps, step)
	}

	step = stepCreateTektonCRDs(params)
	steps = append(steps, step)

	return steps, nil
}

func stepEffectivePipeline(params CRDCreationParameters) syntax.Step {
	args := []string{"--output-dir", "."}
	if params.Context != "" {
		args = append(args, "--context", params.Context)
	}

	step := syntax.Step{
		Name:      createEffectivePipelineStepName,
		Comment:   "Pipeline step creating the effective pipeline configuration",
		Command:   "jx step syntax effective",
		Arguments: args,
	}
	return step
}

func stepCreateTektonCRDs(params CRDCreationParameters) syntax.Step {
	args := []string{"--clone-dir", filepath.Join(tektonBaseDir, params.SourceDir)}
	args = append(args, "--build-number", params.BuildNumber)
	args = append(args, "--trigger", params.Trigger)
	args = append(args, "--service-account", params.ServiceAccount)
	args = append(args, "--source", params.SourceDir)
	if params.Context != "" {
		args = append(args, "--context", params.Context)
	}
	if len(params.Labels) > 0 {
		args = append(args, "--label", strings.Join(params.Labels, ","))
	}
	if len(params.EnvVars) > 0 {
		args = append(args, "--env", strings.Join(params.EnvVars, ","))
	}
	step := syntax.Step{
		Name:      createTektonCRDsStepName,
		Comment:   "Pipeline step to create the Tekton CRDs for the actual pipeline run",
		Command:   "jx step create task",
		Arguments: args,
	}
	return step
}

func determineDefaultStepImage(defaultImage string) string {
	if defaultImage != "" {
		return defaultImage
	}

	return syntax.DefaultContainerImage
}

func buildEnvParams(params CRDCreationParameters) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	envVars = append(envVars, corev1.EnvVar{
		Name:  "JX_LOG_FORMAT",
		Value: "json",
	})

	envVars = append(envVars, corev1.EnvVar{
		Name:  "BUILD_NUMBER",
		Value: params.BuildNumber,
	})

	kind := params.PipelineKind
	if kind != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "PIPELINE_KIND",
			Value: kind,
		})
	}

	context := params.Context
	if context != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "PIPELINE_CONTEXT",
			Value: context,
		})
	}

	gitInfo := params.GitInfo
	u := gitInfo.CloneURL
	if u != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SOURCE_URL",
			Value: u,
		})
	}

	owner := gitInfo.Organisation
	if owner != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "REPO_OWNER",
			Value: owner,
		})
	}

	repo := gitInfo.Name
	if repo != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "REPO_NAME",
			Value: repo,
		})

		// lets keep the APP_NAME environment variable we need for previews
		envVars = append(envVars, corev1.EnvVar{
			Name:  "APP_NAME",
			Value: repo,
		})
	}

	branch := params.Branch
	if branch != "" {
		if kube.GetSliceEnvVar(envVars, "BRANCH_NAME") == nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BRANCH_NAME",
				Value: branch,
			})
		}
	}

	pullRef := params.PullRef
	if pullRef != "" {
		if kube.GetSliceEnvVar(envVars, "PULL_REFS") == nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "PULL_REFS",
				Value: pullRef,
			})
		}
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

	envVars = append(envVars, buildEnvVars(params)...)
	log.Logger().Debugf("step environment variables: %s", util.PrettyPrint(envVars))
	return envVars
}

// TODO: Merge this with step_create_task's setBuildValues equivalent somewhere.
func buildLabels(params CRDCreationParameters) (map[string]string, error) {
	labels := map[string]string{}
	if params.GitInfo != nil {
		labels[tekton.LabelOwner] = params.GitInfo.Organisation
		labels[tekton.LabelRepo] = params.GitInfo.Name
	}
	labels[tekton.LabelBranch] = params.Branch
	if params.Context != "" {
		labels[tekton.LabelContext] = params.Context
	}
	labels[tekton.LabelBuild] = params.BuildNumber

	// add any custom labels
	customLabels, err := util.ExtractKeyValuePairs(params.Labels, "=")
	if err != nil {
		return nil, err
	}
	return util.MergeMaps(labels, customLabels), nil
}

func buildEnvVars(params CRDCreationParameters) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	vars, _ := util.ExtractKeyValuePairs(params.EnvVars, "=")
	for key, value := range vars {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	return envVars
}
