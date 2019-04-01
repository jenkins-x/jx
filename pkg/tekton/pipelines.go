package tekton

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	jxClient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateSourceResource lazily creates a Tekton PipelineResource. This
// function fails if there is already a distinct PipelineResource by the same name.
func CreateSourceResource(tektonClient tektonclient.Interface, ns string, created *v1alpha1.PipelineResource) (*v1alpha1.PipelineResource, error) {
	resourceName := created.Name
	resourceInterface := tektonClient.TektonV1alpha1().PipelineResources(ns)

	_, err2 := resourceInterface.Create(created)
	if err2 == nil {
		return created, nil
	}

	answer, err := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to get PipelineResource %s after failing to create a new one with error %s", resourceName, err2.Error())
	}
	if !reflect.DeepEqual(&created.Spec, &answer.Spec) {
		return nil, errors.Wrapf(err, "Unable to create PipelineResource %s because a PipelineResource with the same name but different values already exists", resourceName)
	}
	return answer, nil
}

// CreateTask lazily creates a Tekton Task. If a Task with the same name already
// exists, this function returns an error.
func CreateTask(tektonClient tektonclient.Interface, ns string, created *v1alpha1.Task) (*v1alpha1.Task, error) {
	resourceName := created.Name
	if resourceName == "" {
		return nil, fmt.Errorf("the Task must have a name")
	}
	resourceInterface := tektonClient.TektonV1alpha1().Tasks(ns)

	answer, err := resourceInterface.Create(created)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Task %s", resourceName)
	}
	return answer, nil
}

// GenerateNextBuildNumber generates a new build number for the given project.
func GenerateNextBuildNumber(jxClient jxClient.Interface, ns string, gitInfo *gits.GitRepository, branch string, duration time.Duration) (string, error) {
	nextBuildNumber := ""
	resourceInterface := jxClient.JenkinsV1().SourceRepositories(ns)
	// TODO: How does SourceRepository handle name overlap?
	sourceRepoName := kube.ToValidName(gitInfo.Organisation + "-" + gitInfo.Name)

	f := func() error {
		sourceRepo, err := kube.GetOrCreateSourceRepository(jxClient, ns, gitInfo.Name, gitInfo.Organisation, gitInfo.ProviderURL())
		if err != nil {
			return errors.Wrapf(err, "Unable to generate next build number for %s/%s", sourceRepoName, branch)
		}
		sourceRepoName = sourceRepo.Name
		if sourceRepo.Annotations == nil {
			sourceRepo.Annotations = make(map[string]string, 1)
		}
		annKey := LastBuildNumberAnnotationPrefix + kube.ToValidName(branch)
		annVal := sourceRepo.Annotations[annKey]
		lastBuildNumber := 0
		if annVal != "" {
			lastBuildNumber, err = strconv.Atoi(annVal)
			if err != nil {
				return errors.Wrapf(err, "Expected number but SourceRepository %s has annotation %s with value %s\n", sourceRepoName, annKey, annVal)
			}
		}
		sourceRepo.Annotations[annKey] = strconv.Itoa(lastBuildNumber + 1)
		if _, err := resourceInterface.Update(sourceRepo); err != nil {
			return err
		}
		nextBuildNumber = sourceRepo.Annotations[annKey]
		return nil
	}

	err := util.Retry(duration, f)
	if err != nil {
		return "", err
	}
	return nextBuildNumber, nil
}

// CreatePipelineRun lazily creates a Tekton PipelineRun.
func CreatePipelineRun(tektonClient tektonclient.Interface, ns string, run *v1alpha1.PipelineRun) (*v1alpha1.PipelineRun, error) {
	resourceName := run.Name
	resourceInterface := tektonClient.TektonV1alpha1().PipelineRuns(ns)

	answer, err := resourceInterface.Create(run)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create PipelineRun %s", resourceName)
	}
	return answer, nil
}

// CreatePipeline lazily creates a Tekton Pipeline for the given git repository,
// branch and context. If a Pipeline with the same name already exists, this
// function returns an error.
func CreatePipeline(tektonClient tektonclient.Interface, ns string, created *v1alpha1.Pipeline) (*v1alpha1.Pipeline, error) {
	resourceName := created.Name
	resourceInterface := tektonClient.TektonV1alpha1().Pipelines(ns)

	answer, err := resourceInterface.Create(created)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Pipeline %s", resourceName)
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
	// TODO: https://github.com/tektoncd/pipeline/issues/481 causes
	// problems since autogenerated container names can end up surpassing 63
	// characters, which is not allowed. Longest known prefix for now is 28
	// chars (build-step-artifact-copy-to-), so we truncate to 35 so the
	// generated container names are no more than 63 chars.
	resourceName := kube.ToValidNameTruncated(dirtyName, 31)
	return resourceName
}
