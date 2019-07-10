package metapipeline

import (
	"fmt"
	"path/filepath"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/prow"
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

	// mergePullRefsStepName is the meta pipeline step name for merging all pull refs into the workspace
	mergePullRefsStepName = "merge-pull-refs"
	// createEffectivePipelineStepName is the meta pipeline step name for the generation of the effective jenkins-x pipeline config
	createEffectivePipelineStepName = "create-effective-pipeline"
	// createTektonCRDsStepName is the meta pipeline step name for the Tekton CRD creation
	createTektonCRDsStepName = "create-tekton-crds"

	tektonBaseDir = "/workspace"
)

// CRDCreationParameters are the parameters needed to create the Tekton CRDs
type CRDCreationParameters struct {
	Namespace        string
	Context          string
	PipelineName     string
	PipelineKind     string
	BuildNumber      string
	GitInfo          gits.GitRepository
	BranchIdentifier string
	PullRef          prow.PullRefs
	SourceDir        string
	PodTemplates     map[string]*corev1.Pod
	ServiceAccount   string
	Labels           []string
	EnvVars          []string
	DefaultImage     string
	Apps             []jenkinsv1.App
	VersionsDir      string
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
	pipeline, tasks, structure, err := parsedPipeline.GenerateCRDs(params.PipelineName, params.BuildNumber, params.Namespace, params.PodTemplates, params.VersionsDir, nil, params.SourceDir, labels, params.DefaultImage)
	if err != nil {
		return nil, err
	}

	resources := []*pipelineapi.PipelineResource{tekton.GenerateSourceRepoResource(params.PipelineName, &params.GitInfo, params.PullRef.BaseBranch)}
	run := tekton.CreatePipelineRun(resources, pipeline.Name, pipeline.APIVersion, labels, params.ServiceAccount, nil, nil)

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

// buildSteps builds the meta pipeline steps.
// The tasks of the meta pipeline are:
// 1) make sure the right commits are merged
// 2) create the effective pipeline and write it to disk
// 3) one step for each extending app
// 4) create Tekton CRDs for the meta pipeline
func buildSteps(params CRDCreationParameters) ([]syntax.Step, error) {
	var steps []syntax.Step

	// 1)
	step := stepMergePullRefs(params.PullRef)
	steps = append(steps, step)

	// 2)
	step = stepEffectivePipeline(params)
	steps = append(steps, step)

	log.Logger().Debugf("creating pipeline steps for extending apps")
	// 3)
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

	// 4)
	step = stepCreateTektonCRDs(params)
	steps = append(steps, step)

	return steps, nil
}

func stepMergePullRefs(pullRefs prow.PullRefs) syntax.Step {
	// we only need to run the merge step in case there is anything to merge
	// Tekton has at this stage the base branch already checked out
	if len(pullRefs.ToMerge) == 0 {
		return stepSkip(mergePullRefsStepName, "Nothing to merge")
	}

	args := []string{"--baseBranch", pullRefs.BaseBranch, "--baseSHA", pullRefs.BaseSha}
	for _, mergeSha := range pullRefs.ToMerge {
		args = append(args, "--sha", mergeSha)
	}

	step := syntax.Step{
		Name:      mergePullRefsStepName,
		Comment:   "Pipeline step merging pull refs",
		Command:   "jx step git merge",
		Arguments: args,
	}
	return step
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
	args = append(args, "--kind", params.PipelineKind)
	for prID := range params.PullRef.ToMerge {
		args = append(args, "--pr-number", prID)
		// there might be a batch build building multiple PRs, in which case we just use the first in this case
		break
	}
	args = append(args, "--service-account", params.ServiceAccount)
	args = append(args, "--source", params.SourceDir)
	args = append(args, "--branch", params.BranchIdentifier)
	args = append(args, "--build-number", params.BuildNumber)
	if params.Context != "" {
		args = append(args, "--context", params.Context)
	}
	for _, l := range params.Labels {
		args = append(args, "--label", l)
	}
	for _, e := range params.EnvVars {
		args = append(args, "--env", e)
	}
	step := syntax.Step{
		Name:      createTektonCRDsStepName,
		Comment:   "Pipeline step to create the Tekton CRDs for the actual pipeline run",
		Command:   "jx step create task",
		Arguments: args,
	}
	return step
}

func stepSkip(stepName string, msg string) syntax.Step {
	skipMsg := fmt.Sprintf("SKIP %s: %s", stepName, msg)
	step := syntax.Step{
		Name:      stepName,
		Comment:   skipMsg,
		Command:   "echo",
		Arguments: []string{fmt.Sprintf("'%s'", skipMsg)},
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

	envVars = append(envVars, corev1.EnvVar{
		Name:  "PIPELINE_KIND",
		Value: params.PipelineKind,
	})

	envVars = append(envVars, corev1.EnvVar{
		Name:  "PULL_REFS",
		Value: params.PullRef.String(),
	})

	context := params.Context
	if context != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "PIPELINE_CONTEXT",
			Value: context,
		})
	}

	gitInfo := params.GitInfo
	envVars = append(envVars, corev1.EnvVar{
		Name:  "SOURCE_URL",
		Value: gitInfo.URL,
	})

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

	branch := params.BranchIdentifier
	if branch != "" {
		if kube.GetSliceEnvVar(envVars, "BRANCH_NAME") == nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BRANCH_NAME",
				Value: branch,
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
	labels[tekton.LabelOwner] = params.GitInfo.Organisation
	labels[tekton.LabelRepo] = params.GitInfo.Name
	labels[tekton.LabelBranch] = params.BranchIdentifier
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
