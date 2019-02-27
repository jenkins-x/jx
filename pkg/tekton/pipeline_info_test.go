package tekton_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	v1fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	tektonv1alpha1 "github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	tektonfake "github.com/knative/build-pipeline/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreatePipelineRunInfo(t *testing.T) {
	t.Parallel()
	ns := "jx"
	testCases := []struct {
		name     string
		expected *tekton.PipelineRunInfo
		prName   string
	}{{
		name: "from-build-pack",
		expected: &tekton.PipelineRunInfo{
			Branch:      "master",
			Build:       "1",
			BuildNumber: 1,
			GitInfo: &gits.GitRepository{
				Host:         "github.com",
				Name:         "jx-demo-qs",
				Organisation: "abayer",
				Project:      "abayer",
				Scheme:       "https",
				URL:          "https://github.com/abayer/jx-demo-qs",
			},
			GitURL:       "https://github.com/abayer/jx-demo-qs",
			Name:         "abayer-jx-demo-qs-master-1",
			Organisation: "abayer",
			Pipeline:     "abayer/jx-demo-qs/master",
			PipelineRun:  "abayer-jx-demo-qs-master-1",
			Repository:   "jx-demo-qs",
			Stages: []*tekton.StageInfo{{
				CreatedTime:    *parseTime(t, "2019-02-21T17:10:48-05:00"),
				FirstStepImage: "gcr.io/k8s-prow/entrypoint@sha256:7c7cd8906ce4982ffee326218e9fc75da2d4896d53cabc9833b9cc8d2d6b2b8f",
				PodName:        "abayer-jx-demo-qs-master-1-build-vhz8d-pod-cd8cba",
				Task:           "abayer-jx-demo-qs-master",
				TaskRun:        "abayer-jx-demo-qs-master-1-build-vhz8d",
			}},
		},
		prName: "abayer-jx-demo-qs-master-1",
	}, {
		name: "from-yaml",
		expected: &tekton.PipelineRunInfo{
			Branch:      "master",
			Build:       "1",
			BuildNumber: 1,
			GitInfo: &gits.GitRepository{
				Host:         "github.com",
				Name:         "js-test-repo",
				Organisation: "abayer",
				Project:      "abayer",
				Scheme:       "https",
				URL:          "https://github.com/abayer/js-test-repo",
			},
			GitURL:       "https://github.com/abayer/js-test-repo",
			Name:         "abayer-js-test-repo-master-1",
			Organisation: "abayer",
			Pipeline:     "abayer/js-test-repo/master",
			PipelineRun:  "abayer-js-test-repo-master-1",
			Repository:   "js-test-repo",
			Stages: []*tekton.StageInfo{{
				Name:           "Build",
				CreatedTime:    *parseTime(t, "2019-02-21T17:02:43-05:00"),
				FirstStepImage: "gcr.io/k8s-prow/entrypoint@sha256:7c7cd8906ce4982ffee326218e9fc75da2d4896d53cabc9833b9cc8d2d6b2b8f",
				PodName:        "abayer-js-test-repo-master-1-build-ttvzf-pod-937200",
				Task:           "abayer-js-test-repo-master-build",
				TaskRun:        "abayer-js-test-repo-master-1-build-ttvzf",
				Parents:        []string{},
			}, {
				Name:           "Second",
				CreatedTime:    *parseTime(t, "2019-02-21T17:03:56-05:00"),
				FirstStepImage: "gcr.io/knative-nightly/github.com/knative/build-pipeline/cmd/bash:v20190221-c649b42c",
				PodName:        "abayer-js-test-repo-master-1-second-9czt5-pod-62d12d",
				Task:           "abayer-js-test-repo-master-second",
				TaskRun:        "abayer-js-test-repo-master-1-second-9czt5",
				Parents:        []string{},
			}},
		},
		prName: "abayer-js-test-repo-master-1",
	}, {
		name: "from-yaml-nested-stages",
		expected: &tekton.PipelineRunInfo{
			Branch:      "nested",
			Build:       "1",
			BuildNumber: 1,
			GitInfo: &gits.GitRepository{
				Host:         "github.com",
				Name:         "js-test-repo",
				Organisation: "abayer",
				Project:      "abayer",
				Scheme:       "https",
				URL:          "https://github.com/abayer/js-test-repo",
			},
			GitURL:       "https://github.com/abayer/js-test-repo",
			Name:         "abayer-js-test-repo-nested-1",
			Organisation: "abayer",
			Pipeline:     "abayer/js-test-repo/nested",
			PipelineRun:  "abayer-js-test-repo-nested-1",
			Repository:   "js-test-repo",
			Stages: []*tekton.StageInfo{{
				Name:    "Parent",
				Parents: []string{},
				Stages: []*tekton.StageInfo{{
					Name:           "Build",
					CreatedTime:    *parseTime(t, "2019-02-21T17:07:36-05:00"),
					FirstStepImage: "gcr.io/k8s-prow/entrypoint@sha256:7c7cd8906ce4982ffee326218e9fc75da2d4896d53cabc9833b9cc8d2d6b2b8f",
					PodName:        "abayer-js-test-repo-nested-1-build-hpqp5-pod-7a19f8",
					Task:           "abayer-js-test-repo-nested-build",
					TaskRun:        "abayer-js-test-repo-nested-1-build-hpqp5",
					Parents:        []string{"Parent"},
				}, {
					Name:           "Second",
					CreatedTime:    *parseTime(t, "2019-02-21T17:08:54-05:00"),
					FirstStepImage: "gcr.io/knative-nightly/github.com/knative/build-pipeline/cmd/bash:v20190221-c649b42c",
					PodName:        "abayer-js-test-repo-nested-1-second-bxxzl-pod-a32406",
					Task:           "abayer-js-test-repo-nested-second",
					TaskRun:        "abayer-js-test-repo-nested-1-second-bxxzl",
					Parents:        []string{"Parent"},
				}},
			}},
		},
		prName: "abayer-js-test-repo-nested-1",
	}, {
		name: "completed-from-yaml",
		expected: nil,
		prName: "abayer-js-test-repo-master-1",
	}, {
		name: "completed-from-build-pack",
		expected: nil,
		prName: "abayer-jx-demo-qs-master-1",
	}}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset(AssertLoadPods(t, tt.name))

			jxObjects := []runtime.Object{AssertLoadPipelineActivity(t, tt.name)}
			structure := AssertLoadPipelineStructure(t, tt.name)
			if structure != nil {
				jxObjects = append(jxObjects, structure)
			}
			jxClient := v1fake.NewSimpleClientset(jxObjects...)

			tektonObjects := []runtime.Object{AssertLoadPipelineRun(t, tt.name), AssertLoadPipeline(t, tt.name)}
			tektonObjects = append(tektonObjects, AssertLoadTasks(t, tt.name))
			tektonObjects = append(tektonObjects, AssertLoadTaskRuns(t, tt.name))
			tektonObjects = append(tektonObjects, AssertLoadPipelineResources(t, tt.name))
			tektonClient := tektonfake.NewSimpleClientset(tektonObjects...)

			pri, err := tekton.CreatePipelineRunInfo(kubeClient, tektonClient, jxClient, ns, tt.prName)
			if err != nil {
				t.Fatalf("Error creating PipelineRunInfo: %s", err)
			}
			if pri == nil {
				if tt.expected != nil {
					t.Errorf("Nil PipelineRunInfo but expected one")
				}
			} else {

				if tt.expected == nil {
					ey, _ := yaml.Marshal(pri)
					t.Logf("%s", ey)
				}

				for _, stage := range pri.Stages {
					scrubPods(stage)
				}

				if d := cmp.Diff(tt.expected, pri); d != "" && tt.expected != nil {
					t.Errorf("Generated PipelineRunInfo did not match expected: %s", d)
				}
			}
		})
	}
}

func AssertLoadPods(t *testing.T, testName string) *corev1.PodList {
	fileName := filepath.Join("test_data", "pipeline_info", testName, "pods.yml")
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

func AssertLoadPipeline(t *testing.T, testName string) *tektonv1alpha1.Pipeline {
	fileName := filepath.Join("test_data", "pipeline_info", testName, "pipeline.yml")
	if tests.AssertFileExists(t, fileName) {
		pipeline := &tektonv1alpha1.Pipeline{}
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

func AssertLoadPipelineRun(t *testing.T, testName string) *tektonv1alpha1.PipelineRun {
	fileName := filepath.Join("test_data", "pipeline_info", testName, "pipelinerun.yml")
	if tests.AssertFileExists(t, fileName) {
		pipelineRun := &tektonv1alpha1.PipelineRun{}
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

func AssertLoadPipelineActivity(t *testing.T, testName string) *v1.PipelineActivity {
	fileName := filepath.Join("test_data", "pipeline_info", testName, "activity.yml")
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

func AssertLoadPipelineStructure(t *testing.T, testName string) *v1.PipelineStructure {
	fileName := filepath.Join("test_data", "pipeline_info", testName, "structure.yml")
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

func AssertLoadTasks(t *testing.T, testName string) *tektonv1alpha1.TaskList {
	fileName := filepath.Join("test_data", "pipeline_info", testName, "tasks.yml")
	if tests.AssertFileExists(t, fileName) {
		taskList := &tektonv1alpha1.TaskList{}
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

func AssertLoadTaskRuns(t *testing.T, testName string) *tektonv1alpha1.TaskRunList {
	fileName := filepath.Join("test_data", "pipeline_info", testName, "taskruns.yml")
	if tests.AssertFileExists(t, fileName) {
		taskRunList := &tektonv1alpha1.TaskRunList{}
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

func AssertLoadPipelineResources(t *testing.T, testName string) *tektonv1alpha1.PipelineResourceList {
	fileName := filepath.Join("test_data", "pipeline_info", testName, "pipelineresources.yml")
	if tests.AssertFileExists(t, fileName) {
		resourceList := &tektonv1alpha1.PipelineResourceList{}
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

func parseTime(t *testing.T, timeString string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, timeString)
	if assert.NoError(t, err, "Failed to parse date %s", timeString) {
		return &parsed
	}
	return nil
}

func scrubPods(s *tekton.StageInfo) {
	s.Pod = nil
	for _, child := range s.Stages {
		scrubPods(child)
	}
	for _, child := range s.Parallel {
		scrubPods(child)
	}
}
