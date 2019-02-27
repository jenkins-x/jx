package tekton

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclient "github.com/knative/build-pipeline/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateOrUpdateSourceResource lazily creates a Tekton Pipeline PipelineResource for the given git repository
func CreateOrUpdateSourceResource(tektonClient tektonclient.Interface, ns string, created *v1alpha1.PipelineResource) (*v1alpha1.PipelineResource, error) {
	resourceName := created.Name
	resourceInterface := tektonClient.TektonV1alpha1().PipelineResources(ns)

	_, err := resourceInterface.Create(created)
	if err == nil {
		return created, nil
	}

	answer, err := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to get PipelineResource %s after failing to create a new one", resourceName)
	}
	if !reflect.DeepEqual(&created.Spec, &answer.Spec) {
		answer.Spec = created.Spec
		answer, err = resourceInterface.Update(answer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to update PipelineResource %s", resourceName)
		}
	}
	return answer, nil
}

// CreateOrUpdateTask lazily creates a Tekton Pipeline Task
func CreateOrUpdateTask(tektonClient tektonclient.Interface, ns string, created *v1alpha1.Task) (*v1alpha1.Task, error) {
	resourceName := created.Name
	if resourceName == "" {
		return nil, fmt.Errorf("the Task must have a name")
	}
	resourceInterface := tektonClient.TektonV1alpha1().Tasks(ns)

	_, err := resourceInterface.Create(created)
	if err == nil {
		return created, nil
	}

	answer, err2 := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err2 != nil {
		return answer, errors.Wrapf(err, "failed to get PipelineResource %s with %v after failing to create a new one", resourceName, err2)
	}
	if !reflect.DeepEqual(&created.Spec, &answer.Spec) || !reflect.DeepEqual(created.Annotations, answer.Annotations) || !reflect.DeepEqual(created.Labels, answer.Labels) {
		answer.Spec = created.Spec
		answer.Labels = util.MergeMaps(answer.Labels, created.Labels)
		answer.Annotations = util.MergeMaps(answer.Annotations, created.Annotations)
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

// CreatePipelineRun lazily creates a Tekton Pipeline Task
func CreatePipelineRun(tektonClient tektonclient.Interface, ns string, pipeline *v1alpha1.Pipeline, run *v1alpha1.PipelineRun, previewVersionPrefix string, duration time.Duration) (*v1alpha1.PipelineRun, error) {
	run.Name = pipeline.Name

	resourceInterface := tektonClient.TektonV1alpha1().PipelineRuns(ns)

	buildNumber := GetLastBuildNumber(pipeline)
	answer := run

	parameters := map[string]string{}

	f := func() error {
		buildNumber++
		buildNumberText := strconv.Itoa(buildNumber)
		run.Labels["build-number"] = buildNumberText

		// lets update the "build_id" parameter if it exists
		for i := range run.Spec.Params {
			switch run.Spec.Params[i].Name {
			case "build_id":
				run.Spec.Params[i].Value = buildNumberText
				parameters["build_id"] = buildNumberText
			case "version":
				if previewVersionPrefix != "" {
					previewVersion := previewVersionPrefix + buildNumberText
					run.Spec.Params[i].Value = previewVersion
					parameters["version"] = previewVersion
				}
			}
		}

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
		err = UpdateLastPipelineBuildNumber(tektonClient, ns, pipeline, buildNumber, parameters, duration)
		if err != nil {
			log.Warnf("Failed to annotate the Pipeline %s with the build number %d: %s", pipeline.Name, buildNumber, err)
		}
		return answer, nil
	}
	return answer, err
}

// UpdateLastPipelineBuildNumber keeps trying to update the last build number annotation on the Pipeline until it succeeds or
// another thread/process beats us to it
func UpdateLastPipelineBuildNumber(tektonClient tektonclient.Interface, ns string, pipeline *v1alpha1.Pipeline, buildNumber int, params map[string]string, duration time.Duration) error {
	f := func() error {
		pipelineInterface := tektonClient.TektonV1alpha1().Pipelines(ns)
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

		// lets override any defaults in the Pipeline
		for i := range current.Spec.Params {
			value := params[current.Spec.Params[i].Name]
			if value != "" {
				current.Spec.Params[i].Default = value
			}
		}

		// lets override any task parameters in the Pipeline
		for i := range current.Spec.Tasks {
			task := &current.Spec.Tasks[i]
			for j := range task.Params {
				param := &task.Params[j]
				value := params[param.Name]
				if value != "" {
					param.Value = value
				}
			}
		}
		_, err = pipelineInterface.Update(current)
		return err
	}
	return util.Retry(duration, f)
}

// CreateOrUpdatePipeline lazily creates a Tekton Pipeline for the given git repository, branch and context
func CreateOrUpdatePipeline(tektonClient tektonclient.Interface, ns string, created *v1alpha1.Pipeline, labels map[string]string) (*v1alpha1.Pipeline, error) {
	resourceName := created.Name
	resourceInterface := tektonClient.TektonV1alpha1().Pipelines(ns)

	answer, err := resourceInterface.Create(created)
	if err == nil {
		return answer, nil
	}

	answer, err = resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to get Pipeline %s after failing to create a new one", resourceName)
	}

	if !reflect.DeepEqual(&created.Spec, &answer.Spec) || !reflect.DeepEqual(created.Labels, answer.Labels) {
		answer.Labels = util.MergeMaps(answer.Annotations, created.Labels, labels)
		answer.Annotations = util.MergeMaps(answer.Annotations, created.Annotations)
		answer.Spec = created.Spec
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
