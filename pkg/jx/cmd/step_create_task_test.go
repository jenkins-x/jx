package cmd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	gits_test "github.com/jenkins-x/jx/pkg/gits/mocks"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/knative/pkg/kmp"
	uuid "github.com/satori/go.uuid"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/tekton/tekton_helpers_test"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateTektonCRDs(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	t.Parallel()

	testData := path.Join("test_data", "step_create_task")
	_, err := os.Stat(testData)
	assert.NoError(t, err)

	packsDir := path.Join(testData, "packs")
	_, err = os.Stat(packsDir)
	assert.NoError(t, err)

	resolver := func(importFile *jenkinsfile.ImportFile) (string, error) {
		dirPath := []string{packsDir, "import_dir", importFile.Import}
		// lets handle cross platform paths in `importFile.File`
		path := append(dirPath, strings.Split(importFile.File, "/")...)
		return filepath.Join(path...), nil
	}

	cases := []struct {
		name           string
		language       string
		repoName       string
		organization   string
		branch         string
		expectingError bool
	}{
		{
			name:         "js_build_pack",
			language:     "javascript",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "build-pack",
		},
		{
			name:         "maven_build_pack",
			language:     "maven",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
		},
		{
			name:         "from_yaml",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
		},
		{
			name:           "no_pipeline_config",
			language:       "none",
			repoName:       "anything",
			organization:   "anything",
			branch:         "anything",
			expectingError: true,
		},
		{
			name:         "per_step_container_build_pack",
			language:     "apps",
			repoName:     "golang-qs-test",
			organization: "abayer",
			branch:       "master",
		},
	}

	k8sObjects := []runtime.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kube.ConfigMapJenkinsDockerRegistry,
				Namespace: "jx",
			},
			Data: map[string]string{
				"docker.registry": "1.2.3.4:5000",
			},
		},
	}
	jxObjects := []runtime.Object{}
	repoOwnerUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	repoOwner := repoOwnerUUID.String()
	repoNameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	repoName := repoNameUUID.String()
	fakeRepo := gits.NewFakeRepository(repoOwner, repoName)
	fakeGitProvider := gits.NewFakeProvider(fakeRepo)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			caseDir := path.Join(testData, tt.name)
			_, err = os.Stat(caseDir)
			assert.NoError(t, err)

			projectConfig, projectConfigFile, err := config.LoadProjectConfig(caseDir)
			assert.NoError(t, err)

			createTask := &cmd.StepCreateTaskOptions{
				Pack:             tt.language,
				NoReleasePrepare: true,
				PipelineKind:     jenkinsfile.PipelineKindRelease,
				PodTemplates:     assertLoadPodTemplates(t),
				GitInfo: &gits.GitRepository{
					Host:         "github.com",
					Name:         tt.repoName,
					Organisation: tt.organization,
				},
				Branch:   tt.branch,
				NoKaniko: true,
				Trigger:  string(pipelineapi.PipelineTriggerTypeManual),
				StepOptions: cmd.StepOptions{
					CommonOptions: &cmd.CommonOptions{
						ServiceAccount: "tekton-bot",
					},
				},
				BuildNumber: "1",
			}
			cmd.ConfigureTestOptionsWithResources(createTask.CommonOptions, k8sObjects, jxObjects, gits_test.NewMockGitter(), fakeGitProvider, helm_test.NewMockHelmer(), nil)

			if tt.language != "none" {
				createTask.SourceName = "source"
			} else {
				createTask.SourceName = "workspace"
			}

			pipeline, tasks, resources, run, structure, err := createTask.GenerateTektonCRDs(packsDir, projectConfig, projectConfigFile, resolver, "jx")
			if tt.expectingError {
				if err == nil {
					t.Fatalf("Expected an error generating CRDs")
				}
			} else {
				if err != nil {
					t.Fatalf("Error generating CRDs: %s", err)
				}

				taskList := &pipelineapi.TaskList{}
				for _, task := range tasks {
					taskList.Items = append(taskList.Items, *task)
				}

				resourceList := &pipelineapi.PipelineResourceList{}
				for _, resource := range resources {
					resourceList.Items = append(resourceList.Items, *resource)
				}

				if d := cmp.Diff(tekton_helpers_test.AssertLoadPipeline(t, caseDir), pipeline); d != "" {
					t.Errorf("Generated Pipeline did not match expected: %s", d)
				}
				if d, _ := kmp.SafeDiff(tekton_helpers_test.AssertLoadTasks(t, caseDir), taskList, cmpopts.IgnoreFields(corev1.ResourceRequirements{}, "Requests")); d != "" {
					t.Errorf("Generated Tasks did not match expected: %s", d)
				}
				if d := cmp.Diff(tekton_helpers_test.AssertLoadPipelineResources(t, caseDir), resourceList); d != "" {
					t.Errorf("Generated PipelineResources did not match expected: %s", d)
				}
				if d := cmp.Diff(tekton_helpers_test.AssertLoadPipelineRun(t, caseDir), run); d != "" {
					t.Errorf("Generated PipelineRun did not match expected: %s", d)
				}
				if d := cmp.Diff(tekton_helpers_test.AssertLoadPipelineStructure(t, caseDir), structure); d != "" {
					t.Errorf("Generated PipelineStructure did not match expected: %s", d)
				}
			}
		})
	}
}

func assertLoadPodTemplates(t *testing.T) map[string]*corev1.Pod {
	fileName := filepath.Join("test_data", "step_create_task", "podTemplates.yml")
	if tests.AssertFileExists(t, fileName) {
		configMap := &corev1.ConfigMap{}
		data, err := ioutil.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, configMap)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				podTemplates := make(map[string]*corev1.Pod)
				for k, v := range configMap.Data {
					pod := &corev1.Pod{}
					if v != "" {
						err := yaml.Unmarshal([]byte(v), pod)
						if assert.NoError(t, err, "Failed to parse pod template") {
							podTemplates[k] = pod
						}
					}
				}
				return podTemplates
			}
		}
	}
	return nil
}
