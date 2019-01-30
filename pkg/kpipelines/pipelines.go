package kpipelines

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	kpipelineclient "github.com/knative/build-pipeline/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

// CreateOrUpdateSourceResource lazily creates a Knative Pipeline PipelineResource for the given git repository
func CreateOrUpdateSourceResource(knativePipelineClient kpipelineclient.Interface, ns string, gitInfo *gits.GitRepository, branch string) (*v1alpha1.PipelineResource, error) {
	if gitInfo == nil {
		return nil, nil
	}

	gitURL := gitInfo.HttpsURL()
	if gitURL == "" {
		return nil, nil
	}
	organisation := gitInfo.Organisation
	name := gitInfo.Name
	resourceName := kube.ToValidName(organisation + "-" + name + "-" + branch)

	resourceInterface := knativePipelineClient.PipelineV1alpha1().PipelineResources(ns)

	created := &v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: v1alpha1.PipelineResourceTypeGit,
			Params: []v1alpha1.Param{
				{
					Name:  "revision",
					Value: branch,
				},
				{
					Name:  "url",
					Value: gitURL,
				},
			},
		},
	}
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

	answer, err := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to get PipelineResource %s after failing to create a new one", resourceName)
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

// CreateOrUpdatePipeline lazily creates a Knative Pipeline for the given git repository, branch and context
func CreateOrUpdatePipeline(knativePipelineClient kpipelineclient.Interface, ns string, gitInfo *gits.GitRepository, branch string, context string, resources []v1alpha1.PipelineDeclaredResource, tasks []v1alpha1.PipelineTask) (*v1alpha1.Pipeline, error) {
	if gitInfo == nil {
		return nil, nil
	}

	gitURL := gitInfo.HttpsURL()
	if gitURL == "" {
		return nil, nil
	}
	resourceName := PipelineResourceName(gitInfo, branch, context)

	resourceInterface := knativePipelineClient.PipelineV1alpha1().Pipelines(ns)

	created := &v1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Spec: v1alpha1.PipelineSpec{
			Resources: resources,
			Tasks:     tasks,
		},
	}
	_, err := resourceInterface.Create(created)
	if err == nil {
		return created, nil
	}

	answer, err := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to get Pipeline %s after failing to create a new one", resourceName)
	}
	copy := *answer

	// lets make sure all the resources and tasks are added
	for _, r1 := range resources {
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
	for _, t1 := range tasks {
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
	if !reflect.DeepEqual(&copy.Spec, &answer.Spec) {
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
