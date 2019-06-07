package tekton_helpers_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// AssertLoadPods reads a file containing a PodList and returns that PodList
func AssertLoadPods(t *testing.T, dir string) *corev1.PodList {
	fileName := filepath.Join(dir, "pods.yml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		t.Fatalf("Error checking if file %s exists: %s", fileName, err)
	}
	if exists {
		podList := &corev1.PodList{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, podList)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return podList
			}

		}
	}
	return &corev1.PodList{}
}

// AssertLoadSinglePod reads a file containing a Pod and returns that Pod
func AssertLoadSinglePod(t *testing.T, dir string) *corev1.Pod {
	fileName := filepath.Join(dir, "pod.yml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		t.Fatalf("Error checking if file %s exists: %s", fileName, err)
	}
	if exists {
		pod := &corev1.Pod{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, pod)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return pod
			}

		}
	}
	return &corev1.Pod{}
}

// AssertLoadPipeline reads a file containing a Pipeline and returns that Pipeline
func AssertLoadPipeline(t *testing.T, dir string) *v1alpha1.Pipeline {
	fileName := filepath.Join(dir, "pipeline.yml")
	if tests.AssertFileExists(t, fileName) {
		pipeline := &v1alpha1.Pipeline{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, pipeline)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return pipeline
			}

		}
	}
	return nil
}

// AssertLoadPipelineRun reads a file containing a PipelineRun and returns that PipelineRun
func AssertLoadPipelineRun(t *testing.T, dir string) *v1alpha1.PipelineRun {
	fileName := filepath.Join(dir, "pipelinerun.yml")
	if tests.AssertFileExists(t, fileName) {
		pipelineRun := &v1alpha1.PipelineRun{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, pipelineRun)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return pipelineRun
			}

		}
	}
	return nil
}

// AssertLoadPipelineActivity reads a file containing a PipelineActivity and returns that PipelineActivity
func AssertLoadPipelineActivity(t *testing.T, dir string) *v1.PipelineActivity {
	fileName := filepath.Join(dir, "activity.yml")
	if tests.AssertFileExists(t, fileName) {
		activity := &v1.PipelineActivity{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, activity)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return activity
			}

		}
	}
	return nil
}

// AssertLoadPipelineStructure reads a file containing a PipelineStructure and returns that PipelineStructure
func AssertLoadPipelineStructure(t *testing.T, dir string) *v1.PipelineStructure {
	fileName := filepath.Join(dir, "structure.yml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		t.Fatalf("Error checking if file %s exists: %s", fileName, err)
	}
	if exists {
		structure := &v1.PipelineStructure{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, structure)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return structure
			}

		}
	}
	return nil
}

// AssertLoadTasks reads a file containing a TaskList and returns that TaskList
func AssertLoadTasks(t *testing.T, dir string) *v1alpha1.TaskList {
	fileName := filepath.Join(dir, "tasks.yml")
	if tests.AssertFileExists(t, fileName) {
		taskList := &v1alpha1.TaskList{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, taskList)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return taskList
			}

		}
	}
	return nil
}

// AssertLoadTaskRuns reads a file containing a TaskRunList and returns that TaskRunList
func AssertLoadTaskRuns(t *testing.T, dir string) *v1alpha1.TaskRunList {
	fileName := filepath.Join(dir, "taskruns.yml")
	if tests.AssertFileExists(t, fileName) {
		taskRunList := &v1alpha1.TaskRunList{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, taskRunList)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return taskRunList
			}

		}
	}
	return nil
}

// AssertLoadPipelineResources reads a file containing a PipelineResourceList and returns that PipelineResourceList
func AssertLoadPipelineResources(t *testing.T, dir string) *v1alpha1.PipelineResourceList {
	fileName := filepath.Join(dir, "pipelineresources.yml")
	if tests.AssertFileExists(t, fileName) {
		resourceList := &v1alpha1.PipelineResourceList{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, resourceList)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return resourceList
			}

		}
	}
	return nil
}
