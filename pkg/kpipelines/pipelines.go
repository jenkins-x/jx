package kpipelines

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	kpipelineclient "github.com/knative/build-pipeline/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strconv"
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
		run.Name =  pipeline.Name + "-" + buildNumberText
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
	resourceName := kube.ToValidName(dirtyName)
	return resourceName
}
