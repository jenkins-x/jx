package kpipelines

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	kpipelineclient "github.com/knative/build-pipeline/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CreateOrUpdateSourceResource lazily creates a Knative Pipeline PipelineResource for the given git repository
func CreateOrUpdateSourceResource(knativePipelineClient kpipelineclient.Interface, ns string, created *v1alpha1.PipelineResource) (*v1alpha1.PipelineResource, error) {
	resourceName := created.Name
	resourceInterface := knativePipelineClient.PipelineV1alpha1().PipelineResources(ns)

	_, err := resourceInterface.Create(created)
	if err == nil {
		return created, nil
	}

	answer, err := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to get PipelineResource %s after failing to create a new one", resourceName)
	}
	copy := *answer
	answer.Spec = created.Spec
	if !reflect.DeepEqual(&copy.Spec, &answer.Spec) {
		answer, err = resourceInterface.Update(answer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to update PipelineResource %s", resourceName)
		}
	}
	return answer, nil
}

// CreateOrUpdateTask lazily creates a Knative Pipeline Task
func CreateOrUpdateTask(knativePipelineClient kpipelineclient.Interface, ns string, created *v1alpha1.Task) (*v1alpha1.Task, error) {
	resourceName := created.Name
	if resourceName == "" {
		return nil, fmt.Errorf("the Task must have a name")
	}
	resourceInterface := knativePipelineClient.PipelineV1alpha1().Tasks(ns)

	_, err := resourceInterface.Create(created)
	if err == nil {
		return created, nil
	}

	answer, err2 := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err2 != nil {
		return answer, errors.Wrapf(err, "failed to get PipelineResource %s with %v after failing to create a new one", resourceName, err2)
	}
	copy := *answer
	answer.Spec = created.Spec
	answer.Labels = util.MergeMaps(answer.Labels, created.Labels)
	answer.Annotations = util.MergeMaps(answer.Annotations, created.Annotations)

	if !reflect.DeepEqual(&copy.Spec, &answer.Spec) || !reflect.DeepEqual(copy.Annotations, answer.Annotations) || !reflect.DeepEqual(copy.Labels, answer.Labels) {
		answer, err = resourceInterface.Update(answer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to update PipelineResource %s", resourceName)
		}
	}
	return answer, nil
}

// GetLastBuildNumber returns the last build number on the Pipeline
func GetLastBuildNumber(pipeline *v1alpha1.Pipeline) int {
	buildNumber := 0
	if pipeline.Annotations != nil {
		ann := pipeline.Annotations[LastBuildNumberAnnotation]
		if ann != "" {
			n, err := strconv.Atoi(ann)
			if err != nil {
				log.Warnf("expected number but Pipeline %s has annotation %s with value %s\n", pipeline.Name, LastBuildNumberAnnotation, ann)
			} else {
				buildNumber = n
			}
		}
	}
	return buildNumber
}

// CreatePipelineRun lazily creates a Knative Pipeline Task
func CreatePipelineRun(knativePipelineClient kpipelineclient.Interface, ns string, pipeline *v1alpha1.Pipeline, run *v1alpha1.PipelineRun, duration time.Duration) (*v1alpha1.PipelineRun, error) {
	run.Name = pipeline.Name

	resourceInterface := knativePipelineClient.PipelineV1alpha1().PipelineRuns(ns)

	buildNumber := GetLastBuildNumber(pipeline)
	answer := run

	f := func() error {
		buildNumber++
		buildNumberText := strconv.Itoa(buildNumber)
		run.Labels["build-number"] = buildNumberText
		run.Name = pipeline.Name + "-" + buildNumberText
		created, err := resourceInterface.Create(run)
		if err == nil {
			answer = created
		}
		return err
	}

	err := util.Retry(duration, f)
	if err == nil {
		// lets try update the pipeline with the new label
		err = UpdateLastPipelineBuildNumber(knativePipelineClient, ns, pipeline, buildNumber, duration)
		if err != nil {
			log.Warnf("Failed to annotate the Pipeline %s with the build number %d: %s", pipeline.Name, buildNumber, err)
		}
		return answer, nil
	}
	return answer, err
}

// UpdateLastPipelineBuildNumber keeps trying to update the last build number annotation on the Pipeline until it succeeds or
// another thread/process beats us to it
func UpdateLastPipelineBuildNumber(knativePipelineClient kpipelineclient.Interface, ns string, pipeline *v1alpha1.Pipeline, buildNumber int, duration time.Duration) error {
	f := func() error {
		pipelineInterface := knativePipelineClient.PipelineV1alpha1().Pipelines(ns)
		current, err := pipelineInterface.Get(pipeline.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		currentBuildNumber := GetLastBuildNumber(current)

		// another PipelineRun has already happened
		if currentBuildNumber > buildNumber {
			return nil
		}
		if current.Annotations == nil {
			current.Annotations = map[string]string{}
		}
		current.Annotations[LastBuildNumberAnnotation] = strconv.Itoa(buildNumber)
		_, err = pipelineInterface.Update(current)
		return err
	}
	return util.Retry(duration, f)
}

// CreateOrUpdatePipeline lazily creates a Knative Pipeline for the given git repository, branch and context
func CreateOrUpdatePipeline(knativePipelineClient kpipelineclient.Interface, ns string, created *v1alpha1.Pipeline, labels map[string]string) (*v1alpha1.Pipeline, error) {
	resourceName := created.Name
	resourceInterface := knativePipelineClient.PipelineV1alpha1().Pipelines(ns)

	answer, err := resourceInterface.Create(created)
	if err == nil {
		return answer, nil
	}

	answer, err = resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to get Pipeline %s after failing to create a new one", resourceName)
	}
	copy := *answer

	answer.Labels = util.MergeMaps(answer.Labels, labels)

	// lets make sure all the resources and tasks are added
	for _, r1 := range created.Spec.Resources {
		found := false
		for _, r2 := range answer.Spec.Resources {
			if reflect.DeepEqual(&r1, &r2) {
				found = true
				break
			}
		}
		if !found {
			answer.Spec.Resources = append(answer.Spec.Resources, r1)
		}
	}
	for _, t1 := range created.Spec.Tasks {
		found := false
		for _, t2 := range answer.Spec.Tasks {
			if reflect.DeepEqual(&t1, &t2) {
				found = true
				break
			}
		}
		if !found {
			answer.Spec.Tasks = append(answer.Spec.Tasks, t1)
		}
	}
	if !reflect.DeepEqual(&copy.Spec, &answer.Spec) || !reflect.DeepEqual(copy.Labels, answer.Labels) {
		answer, err = resourceInterface.Update(answer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to update Pipeline %s", resourceName)
		}
	}
	return answer, nil
}

// PipelineResourceName returns the pipeline resource name for the given git repository, branch and context
func PipelineResourceName(gitInfo *gits.GitRepository, branch string, context string) string {
	organisation := gitInfo.Organisation
	name := gitInfo.Name
	dirtyName := organisation + "-" + name + "-" + branch
	if context != "" {
		dirtyName += "-" + context
	}
	// TODO: https://github.com/knative/build-pipeline/issues/481 causes
	// problems since autogenerated container names can end up surpassing 63
	// characters, which is not allowed. Longest known prefix for now is 28
	// chars (build-step-artifact-copy-to-), so we truncate to 35 so the
	// generated container names are no more than 63 chars.
	resourceName := kube.ToValidNameTruncated(dirtyName, 35)
	return resourceName
}

var randReader = rand.Reader

func scopedEnv(newEnv []v1.PipelineStructureEnvVar, parentEnv []corev1.EnvVar) []corev1.EnvVar {
	if len(parentEnv) == 0 && len(newEnv) == 0 {
		return nil
	}
	envMap := make(map[string]corev1.EnvVar)

	for _, e := range parentEnv {
		envMap[e.Name] = e
	}

	for _, e := range newEnv {
		envMap[e.Name] = corev1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		}
	}

	env := make([]corev1.EnvVar, 0, len(envMap))

	// Avoid nondeterministic results by sorting the keys and appending vars in that order.
	var envVars []string
	for k := range envMap {
		envVars = append(envVars, k)
	}
	sort.Strings(envVars)

	for _, envVar := range envVars {
		env = append(env, envMap[envVar])
	}

	return env
}

// +k8s:openapi-gen=false
type transformedStage struct {
	Stage *v1.PipelineStructureStage
	// Only one of Sequential, Parallel, and Task is non-empty
	Sequential []*transformedStage
	Parallel   []*transformedStage
	Task       *v1alpha1.Task
	// PipelineTask is non-empty only if Task is non-empty, but it is populated
	// after Task is populated so the reverse is not true.
	PipelineTask *v1alpha1.PipelineTask
	// The depth of this stage in the full tree of stages
	Depth int8
	// The parallel or sequntial stage enclosing this stage, or nil if this stage is at top level
	EnclosingStage *transformedStage
	// The stage immediately before this stage at the same depth, or nil if there is no such stage
	PreviousSiblingStage *transformedStage
	// TODO: Add the equivalent reverse relationship
}

func (ts transformedStage) isSequential() bool {
	return len(ts.Sequential) > 0
}

func (ts transformedStage) isParallel() bool {
	return len(ts.Parallel) > 0
}

func (ts transformedStage) getLinearTasks() []*v1alpha1.Task {
	if ts.isSequential() {
		var tasks []*v1alpha1.Task
		for _, seqTs := range ts.Sequential {
			tasks = append(tasks, seqTs.getLinearTasks()...)
		}
		return tasks
	} else if ts.isParallel() {
		var tasks []*v1alpha1.Task
		for _, parTs := range ts.Parallel {
			tasks = append(tasks, parTs.getLinearTasks()...)
		}
		return tasks
	} else {
		return []*v1alpha1.Task{ts.Task}
	}
}

// If the workspace is nil, return the parent's workspace, otherwise return the workspace
func (ts *transformedStage) computeWorkspace(parentWorkspace string) {
	if ts.Stage.Options == nil || ts.Stage.Options.Workspace == nil {
		if ts.Stage.Options == nil {
			ts.Stage.Options = &v1.PipelineStructureStageOptions{}
		}
		ts.Stage.Options.Workspace = &parentWorkspace
	}
}

func stageToTask(s *v1.PipelineStructureStage, pipelineIdentifier string, buildIdentifier string, namespace string, wsPath string, parentEnv []corev1.EnvVar, parentAgent *v1.PipelineStructureAgent, parentWorkspace string, suffix string, depth int8, enclosingStage *transformedStage, previousSiblingStage *transformedStage, podTemplates map[string]*corev1.Pod) (*transformedStage, error) {
	if len(s.Post) != 0 {
		return nil, errors.New("post on stages not yet supported")
	}

	if s.Options != nil {
		o := s.Options
		if o.Timeout != nil {
			return nil, errors.New("Timeout on stage not yet supported")
		}
		if o.Retry > 0 {
			return nil, errors.New("Retry on stage not yet supported")
		}
		if o.Stash != nil {
			return nil, errors.New("Stash on stage not yet supported")
		}
		if o.Unstash != nil {
			return nil, errors.New("Unstash on stage not yet supported")
		}
	}

	if depth > 0 {
		s.Depth = depth
	}
	if enclosingStage != nil {
		s.Parent = &enclosingStage.Stage.Name
	}

	env := scopedEnv(s.Environment, parentEnv)

	agent := s.Agent

	if agent == nil {
		agent = parentAgent
	}

	stepCounter := 0

	if len(s.Steps) > 0 {
		if suffix == "" {
			// Generate a short random hex string.
			b, err := ioutil.ReadAll(io.LimitReader(randReader, 3))
			if err != nil {
				return nil, err
			}
			suffix = hex.EncodeToString(b)
		}

		t := &v1alpha1.Task{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.PipelineAPIVersion,
				Kind:       "Task",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      v1.MangleToRfc1035Label(fmt.Sprintf("%s-%s", pipelineIdentifier, s.Name), ""),
				Labels:    map[string]string{v1.LabelStageName: s.Name},
			},
		}
		t.SetDefaults()

		ws := &v1alpha1.TaskResource{
			Name: "workspace",
			Type: v1alpha1.PipelineResourceTypeGit,
		}

		if wsPath != "" {
			ws.TargetPath = wsPath
		}

		t.Spec.Inputs = &v1alpha1.Inputs{
			Resources: []v1alpha1.TaskResource{*ws,
				{
					Name: "temp-ordering-resource",
					Type: v1alpha1.PipelineResourceTypeImage,
				},
			},
		}

		t.Spec.Outputs = &v1alpha1.Outputs{
			Resources: []v1alpha1.TaskResource{
				{
					Name: "workspace",
					Type: v1alpha1.PipelineResourceTypeGit,
				},
				{
					Name: "temp-ordering-resource",
					Type: v1alpha1.PipelineResourceTypeImage,
				},
			},
		}

		// We don't want to dupe volumes for the Task if there are multiple steps
		volumes := make(map[string]corev1.Volume)
		for _, step := range s.Steps {
			actualSteps, stepVolumes, newCounter, err := generateSteps(step, *agent.Image, env, podTemplates, stepCounter)
			if err != nil {
				return nil, err
			}

			stepCounter = newCounter

			t.Spec.Steps = append(t.Spec.Steps, actualSteps...)
			for k, v := range stepVolumes {
				volumes[k] = v
			}
		}
		for _, volume := range volumes {
			t.Spec.Volumes = append(t.Spec.Volumes, volume)
		}
		ts := transformedStage{Stage: s, Task: t, Depth: depth, EnclosingStage: enclosingStage, PreviousSiblingStage: previousSiblingStage}
		ts.computeWorkspace(parentWorkspace)
		return &ts, nil
	}

	if len(s.Stages) > 0 {
		var tasks []*transformedStage
		ts := transformedStage{Stage: s, Depth: depth, EnclosingStage: enclosingStage, PreviousSiblingStage: previousSiblingStage}
		ts.computeWorkspace(parentWorkspace)

		for i, nested := range s.Stages {
			nestedWsPath := ""
			if wsPath != "" && i == 0 {
				nestedWsPath = wsPath
			}
			var nestedPreviousSibling *transformedStage
			if i > 0 {
				nestedPreviousSibling = tasks[i-1]
			}
			nestedTask, err := stageToTask(nested, pipelineIdentifier, buildIdentifier, namespace, nestedWsPath, env, agent, *ts.Stage.Options.Workspace, suffix, depth+1, &ts, nestedPreviousSibling, podTemplates)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, nestedTask)
		}
		ts.Sequential = tasks

		return &ts, nil
	}

	if len(s.Parallel) > 0 {
		var tasks []*transformedStage
		ts := transformedStage{Stage: s, Depth: depth, EnclosingStage: enclosingStage, PreviousSiblingStage: previousSiblingStage}
		ts.computeWorkspace(parentWorkspace)

		for i, nested := range s.Parallel {
			nestedWsPath := ""
			if wsPath != "" && i == 0 {
				nestedWsPath = wsPath
			}
			nestedTask, err := stageToTask(nested, pipelineIdentifier, buildIdentifier, namespace, nestedWsPath, env, agent, *ts.Stage.Options.Workspace, suffix, depth+1, &ts, nil, podTemplates)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, nestedTask)
		}
		ts.Parallel = tasks

		return &ts, nil
	}

	return nil, errors.New("no steps, sequential stages, or parallel stages")
}

func generateSteps(step v1.PipelineStructureStep, inheritedAgent string, env []corev1.EnvVar, podTemplates map[string]*corev1.Pod, stepCounter int) ([]corev1.Container, map[string]corev1.Volume, int, error) {
	volumes := make(map[string]corev1.Volume)
	var steps []corev1.Container

	stepImage := inheritedAgent
	if step.Agent != nil {
		stepImage = *step.Agent.Image
	}

	if step.Command != nil {
		var c corev1.Container

		if podTemplates != nil && podTemplates[stepImage] != nil {
			podTemplate := podTemplates[stepImage]
			containers := podTemplate.Spec.Containers
			for _, volume := range podTemplate.Spec.Volumes {
				volumes[volume.Name] = volume
			}
			c = containers[0]
			cmdStr := *step.Command
			if len(step.Args) > 0 {
				cmdStr += " " + strings.Join(step.Args, " ")
			}
			c.Args = []string{cmdStr}
			c.WorkingDir = "/workspace/workspace"
		} else {
			c = corev1.Container{
				Image:   stepImage,
				Command: []string{*step.Command},
				Args:    step.Args,
				// TODO: Better paths
				WorkingDir: "/workspace/workspace",
			}
		}
		stepCounter++
		c.Name = "step" + strconv.Itoa(1+stepCounter)

		c.Stdin = false
		c.TTY = false

		c.Env = env

		steps = append(steps, c)
	} else if step.Loop != nil {
		for _, v := range step.Loop.Values {
			loopEnv := scopedEnv([]v1.PipelineStructureEnvVar{{Name: step.Loop.Variable, Value: v}}, env)

			for _, s := range step.Loop.Steps {
				loopSteps, loopVolumes, loopCounter, loopErr := generateSteps(s, stepImage, loopEnv, podTemplates, stepCounter)
				if loopErr != nil {
					return nil, nil, loopCounter, loopErr
				}

				// Bump the step counter to what we got from the loop
				stepCounter = loopCounter

				// Add the loop-generated steps
				steps = append(steps, loopSteps...)

				// Add any new volumes that may have shown up
				for k, v := range loopVolumes {
					volumes[k] = v
				}
			}
		}
	} else {
		return nil, nil, stepCounter, errors.New("syntactic sugar steps not yet supported")
	}

	return steps, volumes, stepCounter, nil
}

// GenerateCRDs translates the Pipeline structure into the corresponding Pipeline and Task CRDs
func GenerateCRDs(j *v1.PipelineStructure, pipelineIdentifier string, buildIdentifier string, namespace string, suffix string, podTemplates map[string]*corev1.Pod) (*v1alpha1.Pipeline, []*v1alpha1.Task, *v1.PipelineStructure, error) {
	if len(j.Post) != 0 {
		return nil, nil, nil, errors.New("Post at top level not yet supported")
	}

	if j.Options != nil {
		o := j.Options
		if o.Retry > 0 {
			return nil, nil, nil, errors.New("Retry at top level not yet supported")
		}
	}

	if suffix == "" {
		// Generate a short random hex string.
		b, err := ioutil.ReadAll(io.LimitReader(randReader, 3))
		if err != nil {
			return nil, nil, nil, err
		}
		suffix = hex.EncodeToString(b)
	}

	p := &v1alpha1.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.PipelineAPIVersion,
			Kind:       "Pipeline",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s", pipelineIdentifier),
		},
		Spec: v1alpha1.PipelineSpec{
			Resources: []v1alpha1.PipelineDeclaredResource{
				{
					Name: pipelineIdentifier,
					Type: v1alpha1.PipelineResourceTypeGit,
				},
				{
					// TODO: Switch from this kind of hackish approach to non-resource-based dependencies once they land.
					Name: "temp-ordering-resource",
					Type: v1alpha1.PipelineResourceTypeImage,
				},
			},
		},
	}

	p.SetDefaults()

	var previousStage *transformedStage

	var tasks []*v1alpha1.Task

	baseEnv := j.ToContainerEnvVars()

	for _, s := range j.Stages {
		wsPath := ""
		if len(tasks) == 0 {
			wsPath = "workspace"
		}
		stage, err := stageToTask(s, pipelineIdentifier, buildIdentifier, namespace, wsPath, baseEnv, j.Agent, "default", suffix, 0, nil, previousStage, podTemplates)
		if err != nil {
			return nil, nil, nil, err
		}
		previousStage = stage

		tasks = append(tasks, stage.getLinearTasks()...)
		p.Spec.Tasks = append(p.Spec.Tasks, createPipelineTasks(stage, pipelineIdentifier)...)
	}

	return p, tasks, j, nil
}

func createPipelineTasks(stage *transformedStage, pipelineIdentifier string) []v1alpha1.PipelineTask {
	if stage.isSequential() {
		var pTasks []v1alpha1.PipelineTask
		for _, nestedStage := range stage.Sequential {
			pTasks = append(pTasks, createPipelineTasks(nestedStage, pipelineIdentifier)...)
		}
		return pTasks
	} else if stage.isParallel() {
		var pTasks []v1alpha1.PipelineTask
		for _, nestedStage := range stage.Parallel {
			pTasks = append(pTasks, createPipelineTasks(nestedStage, pipelineIdentifier)...)
		}
		return pTasks
	} else {
		pTask := v1alpha1.PipelineTask{
			Name: stage.Stage.TaskName(), // TODO: What should this actually be named?
			TaskRef: v1alpha1.TaskRef{
				Name: stage.Task.Name,
			},
		}

		_, provider := findWorkspaceProvider(stage, stage.getEnclosing(0))
		var previousStageNames []string
		for _, previousStage := range findPreviousNonBlockStages(*stage) {
			previousStageNames = append(previousStageNames, previousStage.PipelineTask.Name)
		}
		pTask.Resources = &v1alpha1.PipelineTaskResources{
			Inputs: []v1alpha1.PipelineTaskInputResource{
				{
					Name:     "workspace",
					Resource: pipelineIdentifier,
					From:     provider,
				},
				{
					// TODO: Switch from this kind of hackish approach to non-resource-based dependencies once they land.
					Name:     "temp-ordering-resource",
					Resource: "temp-ordering-resource",
					From:     previousStageNames,
				},
			},
			Outputs: []v1alpha1.PipelineTaskOutputResource{
				{
					Name:     "workspace",
					Resource: pipelineIdentifier,
				},
				{
					// TODO: Switch from this kind of hackish approach to non-resource-based dependencies once they land.
					Name:     "temp-ordering-resource",
					Resource: "temp-ordering-resource",
				},
			},
		}
		stage.PipelineTask = &pTask

		return []v1alpha1.PipelineTask{pTask}
	}
}

// Looks for the most recent Task using the desired workspace that was not in the
// same parallel stage and returns the name of the corresponding Task.
func findWorkspaceProvider(stage, sibling *transformedStage) (bool, []string) {
	if *stage.Stage.Options.Workspace == "empty" {
		return true, nil
	}

	for sibling != nil {
		if sibling.isSequential() {
			found, provider := findWorkspaceProvider(stage, sibling.Sequential[len(sibling.Sequential)-1])
			if found {
				return true, provider
			}
		} else if sibling.isParallel() {
			// We don't want to use a workspace from a parallel stage outside of that stage,
			// but we do need to descend inwards in case stage is in that same stage.
			if stage.getEnclosing(sibling.Depth) == sibling {
				for _, nested := range sibling.Parallel {
					// Pick the parallel branch that has stage
					if stage.getEnclosing(nested.Depth) == nested {
						found, provider := findWorkspaceProvider(stage, nested)
						if found {
							return true, provider
						}
					}
				}
			}
			// TODO: What to do about custom workspaces? Check for erroneous uses specially?
			// Allow them if only one of the parallel tasks uses the same resource?
		} else if sibling.PipelineTask != nil {
			if *sibling.Stage.Options.Workspace == *stage.Stage.Options.Workspace {
				return true, []string{sibling.PipelineTask.Name}
			}
		} else {
			// We are in a sequential stage and sibling has not had its PipelineTask created.
			// Check the task before it so we don't use a workspace of a later task.
		}
		sibling = sibling.PreviousSiblingStage
	}

	return false, nil
}

// Find the end tasks for this stage, traversing down to the end stages of any
// nested sequential or parallel stages as well.
func findEndStages(stage transformedStage) []*transformedStage {
	if stage.isSequential() {
		return findEndStages(*stage.Sequential[len(stage.Sequential)-1])
	} else if stage.isParallel() {
		var endTasks []*transformedStage
		for _, pStage := range stage.Parallel {
			endTasks = append(endTasks, findEndStages(*pStage)...)
		}
		return endTasks
	} else {
		return []*transformedStage{&stage}
	}
}

// Find the tasks that run immediately before this stage, not including
// sequential or parallel wrapper stages.
func findPreviousNonBlockStages(stage transformedStage) []*transformedStage {
	if stage.PreviousSiblingStage != nil {
		return findEndStages(*stage.PreviousSiblingStage)
	} else if stage.EnclosingStage != nil {
		return findPreviousNonBlockStages(*stage.EnclosingStage)
	} else {
		return []*transformedStage{}
	}
}

// Return the stage that encloses this stage at the given depth, or nil if there is no such stage.
// Depth must be >= 0. Returns the stage itself if depth == stage.Depth
func (ts *transformedStage) getEnclosing(depth int8) *transformedStage {
	if ts.Depth == depth {
		return ts
	} else if ts.EnclosingStage == nil {
		return nil
	} else {
		return ts.EnclosingStage.getEnclosing(depth)
	}
}

// Return the first stage that will execute before this stage
// Depth must be >= 0
func (ts transformedStage) getClosestAncestor() *transformedStage {
	if ts.PreviousSiblingStage != nil {
		return ts.PreviousSiblingStage
	} else if ts.EnclosingStage == nil {
		return nil
	} else {
		return ts.EnclosingStage.getClosestAncestor()
	}
}
