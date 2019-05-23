package metapipeline

import (
	"fmt"
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
	Revision       string
	Labels         map[string]string
	EnvVars        []corev1.EnvVar
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

	// TODO verify 'nil' for taskParams
	pipeline, tasks, structure, err := parsedPipeline.GenerateCRDs(params.PipelineName, params.BuildNumber, params.Namespace, params.PodTemplates, nil, params.SourceDir)
	if err != nil {
		return nil, err
	}

	pipelineResourceName := tekton.PipelineResourceName(params.GitInfo, params.Branch, params.Context)
	resources := []*pipelineapi.PipelineResource{tekton.GenerateSourceRepoResource(pipelineResourceName, params.GitInfo, params.Revision)}
	// TODO verify 'nil' for pipelineParams
	run := tekton.CreatePipelineRun(resources, pipeline.Name, pipeline.APIVersion, params.Labels, params.Trigger, params.ServiceAccount, nil)

	tektonCRDs, err := tekton.NewCRDWrapper(pipeline, tasks, resources, structure, run)
	if err != nil {
		return nil, err
	}

	return tektonCRDs, nil
}

// createPipeline builds the parsed/typed pipeline which servers as input for the Tekton CRD creation.
func createPipeline(params CRDCreationParameters) (*syntax.ParsedPipeline, error) {
	steps, err := BuildExtensionSteps(params.Apps)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create app extending pipeline steps")
	}

	stage := syntax.Stage{
		Name:  appExtensionStageName,
		Steps: steps,
		Agent: &syntax.Agent{
			// TODO Agent does not seem to be used anywhere
			Image: "unused",
		},
	}

	parsedPipeline := &syntax.ParsedPipeline{
		Stages: []syntax.Stage{stage},
	}

	env := buildEnvParams(params)
	parsedPipeline.AddContainerEnvVarsToPipeline(env)

	return parsedPipeline, nil
}

// GetExtendingApps returns the lst of apps which are installed in the cluster registered for extending the pipeline.
// An app registers its interest in extending the pipeline by having the 'pipeline-extension' label set.
func GetExtendingApps(jxClient versioned.Interface, namespace string) ([]jenkinsv1.App, error) {
	listOptions := metav1.ListOptions{}
	listOptions.LabelSelector = fmt.Sprintf(apps.AppTypeLabel+" in (%s)", apps.PipelineExtension)
	appsList, err := jxClient.JenkinsV1().Apps(namespace).List(listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error  retrieving pipeline contributor apps")
	}
	return appsList.Items, nil
}

// BuildExtensionSteps builds a step (container) for each of the apps specified.
func BuildExtensionSteps(extendingApps []jenkinsv1.App) ([]syntax.Step, error) {
	var steps []syntax.Step

	// TODO - Add step to create effective pipeline. This way apps will always have a pipeline config to load and work with,
	// TODO - even if the source uses an implicit pipeline config
	// TODO - In case of a release we might want to add a task to create the tag

	log.Debugf("creating pipeline steps for extending apps")
	for _, app := range extendingApps {
		if app.Spec.PipelineExtension == nil {
			log.Warnf("Skipping app %s in meta pipeline. It contains label %s with value %s, but does not contain PipelineExtension fields.", app.Name, apps.AppTypeLabel, apps.PipelineExtension)
			continue
		}

		extension := app.Spec.PipelineExtension
		step := syntax.Step{
			Name:      extension.Name,
			Image:     extension.Image,
			Command:   extension.Command,
			Arguments: extension.Args,
		}

		log.Debugf("App %s contributes with step %s", app.Name, util.PrettyPrint(step))
		steps = append(steps, step)
	}

	// TODO - Add step which does call the 'step create task' functionality to build the actual build pipeline
	return steps, nil
}

func buildEnvParams(params CRDCreationParameters) []corev1.EnvVar {
	var envVars []corev1.EnvVar

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

	envVars = append(envVars, params.EnvVars...)
	log.Debugf("step environment variables: %s", util.PrettyPrint(envVars))
	return envVars
}
