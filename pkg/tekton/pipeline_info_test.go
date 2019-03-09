package tekton_test

import (
	"path"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/tekton/tekton_helpers_test"
	tektonfake "github.com/knative/build-pipeline/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreatePipelineRunInfo(t *testing.T) {
	t.Parallel()
	ns := "jx"
	testCases := []struct {
		name     string
		expected *tekton.PipelineRunInfo
		prName   string
	}{{
		name: "from-build-pack-init-containers",
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
				Name:           syntax.DefaultStageNameForBuildPack,
				CreatedTime:    *parseTime(t, "2019-02-21T17:10:48-05:00"),
				FirstStepImage: "gcr.io/k8s-prow/entrypoint@sha256:7c7cd8906ce4982ffee326218e9fc75da2d4896d53cabc9833b9cc8d2d6b2b8f",
				PodName:        "abayer-jx-demo-qs-master-1-build-vhz8d-pod-cd8cba",
				Task:           "abayer-jx-demo-qs-master",
				TaskRun:        "abayer-jx-demo-qs-master-1-build-vhz8d",
				Parents:        []string{},
			}},
		},
		prName: "abayer-jx-demo-qs-master-1",
	}, {
		name: "from-yaml-init-containers",
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
		name: "from-yaml-nested-stages-init-containers",
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
				CreatedTime:    *parseTime(t, "2019-03-05T15:06:13-05:00"),
				FirstStepImage: "jenkinsxio/builder-nodejs:0.1.263",
				PodName:        "abayer-js-test-repo-master-1-build-jmcbd-pod-a726d6",
				Task:           "abayer-js-test-repo-master-build",
				TaskRun:        "abayer-js-test-repo-master-1-build-jmcbd",
				Parents:        []string{},
			}, {
				Name:           "Second",
				CreatedTime:    *parseTime(t, "2019-03-05T15:07:05-05:00"),
				FirstStepImage: "us.gcr.io/abayer-jx-experiment/bash-1155d67b477d7c4e2f7998b1fc6b4e43@sha256:ad8c6fffadb5f2723fe8a4aa3ac7f4ac091e1fe14b1badec7418c3705911af3c",
				PodName:        "abayer-js-test-repo-master-1-second-wglk8-pod-762f8d",
				Task:           "abayer-js-test-repo-master-second",
				TaskRun:        "abayer-js-test-repo-master-1-second-wglk8",
				Parents:        []string{},
			}},
		},
		prName: "abayer-js-test-repo-master-1",
	}, {
		name:     "completed-from-yaml",
		expected: nil,
		prName:   "abayer-js-test-repo-master-1",
	}, {
		name:     "completed-from-build-pack",
		expected: nil,
		prName:   "abayer-jx-demo-qs-master-1",
	}}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			testCaseDir := path.Join("test_data", "pipeline_info", tt.name)

			jxObjects := []runtime.Object{tekton_helpers_test.AssertLoadPipelineActivity(t, testCaseDir)}
			structure := tekton_helpers_test.AssertLoadPipelineStructure(t, testCaseDir)
			if structure != nil {
				jxObjects = append(jxObjects, structure)
			}

			tektonObjects := []runtime.Object{tekton_helpers_test.AssertLoadPipelineRun(t, testCaseDir), tekton_helpers_test.AssertLoadPipeline(t, testCaseDir)}
			tektonObjects = append(tektonObjects, tekton_helpers_test.AssertLoadTasks(t, testCaseDir))
			tektonObjects = append(tektonObjects, tekton_helpers_test.AssertLoadTaskRuns(t, testCaseDir))
			tektonObjects = append(tektonObjects, tekton_helpers_test.AssertLoadPipelineResources(t, testCaseDir))
			tektonClient := tektonfake.NewSimpleClientset(tektonObjects...)

			podList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)

			pr, err := tektonClient.TektonV1alpha1().PipelineRuns(ns).Get(tt.prName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Error fetching PipelineRun: %s", err)
			}
			pri, err := tekton.CreatePipelineRunInfo(tt.prName, podList, structure, pr)
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
