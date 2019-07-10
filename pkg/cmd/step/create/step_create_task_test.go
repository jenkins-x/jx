package create

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/knative/pkg/kmp"
	"github.com/satori/go.uuid"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/tekton/tekton_helpers_test"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testCase struct {
	name           string
	language       string
	repoName       string
	organization   string
	branch         string
	kind           string
	expectingError bool
	useKaniko      bool
}

func TestGenerateTektonCRDs(t *testing.T) {
	t.Parallel()

	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	testData := path.Join("test_data", "step_create_task")
	_, err := os.Stat(testData)
	assert.NoError(t, err)

	testVersionsDir := path.Join(testData, "stable_versions")
	packsDir := path.Join(testData, "packs")
	_, err = os.Stat(packsDir)
	assert.NoError(t, err)

	rand.Seed(12345)

	resolver := func(importFile *jenkinsfile.ImportFile) (string, error) {
		dirPath := []string{packsDir, "import_dir", importFile.Import}
		// lets handle cross platform paths in `importFile.File`
		path := append(dirPath, strings.Split(importFile.File, "/")...)
		return filepath.Join(path...), nil
	}

	cases := []testCase{
		{
			name:         "js_build_pack",
			language:     "javascript",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "build-pack",
			kind:         "release",
			useKaniko:    true,
		},
		{
			name:         "maven_build_pack",
			language:     "maven",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
			useKaniko:    false,
		},
		{
			name:         "from_yaml",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:           "no_pipeline_config",
			language:       "none",
			repoName:       "anything",
			organization:   "anything",
			branch:         "anything",
			kind:           "release",
			expectingError: true,
		},
		{
			name:         "per_step_container_build_pack",
			language:     "apps",
			repoName:     "golang-qs-test",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
		},
		{
			name:         "kaniko_entrypoint",
			language:     "none",
			repoName:     "jx",
			organization: "jenkins-x",
			branch:       "fix-kaniko-special-casing",
			kind:         "pullrequest",
		},
		{
			name:         "set-agent-container-with-agentless-build-pack",
			language:     "no-default-agent",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "no-default-agent",
			kind:         "release",
		},
		{
			name:         "override-agent-container-with-build-pack",
			language:     "override-default-agent",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "override-default-agent",
			kind:         "release",
		},
		{
			name:         "override-steps",
			language:     "maven",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
			useKaniko:    false,
		},
		{
			name:         "override_block_step",
			language:     "apps",
			repoName:     "golang-qs-test",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
		},
		{
			name:         "loop-in-buildpack-syntax",
			language:     "maven",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
			useKaniko:    false,
		},
		{
			name:         "containeroptions-on-pipelineconfig",
			language:     "maven",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
			useKaniko:    false,
		},
		{
			name:         "default-in-jenkins-x-yml",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:         "default-in-buildpack",
			language:     "default-pipeline",
			repoName:     "golang-qs-test",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
		},
		{
			name:         "add-env-to-default-in-buildpack",
			language:     "default-pipeline",
			repoName:     "golang-qs-test",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
		},
		{
			name:         "override-default-in-buildpack",
			language:     "default-pipeline",
			repoName:     "golang-qs-test",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
		},
		{
			name:         "override-default-in-jenkins-x-yml",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:         "remove-stage",
			language:     "maven",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
		},
		{
			name:           "remove-pipeline",
			language:       "none",
			repoName:       "anything",
			organization:   "anything",
			branch:         "anything",
			kind:           "pullRequest",
			expectingError: true,
		},
		{
			name:         "remove-stage-from-jenkins-x-yml",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:           "remove-pipeline-from-jenkins-x-yml",
			language:       "none",
			repoName:       "anything",
			organization:   "anything",
			branch:         "anything",
			kind:           "pullRequest",
			expectingError: true,
		},
		{
			name:         "replace-stage-steps",
			language:     "maven",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
		},
		{
			name:         "append-and-prepend-stage-steps",
			language:     "maven",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
			useKaniko:    false,
		},
		{
			name:         "replace-stage-steps-in-jenkins-x-yml",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:         "append-and-prepend-stage-steps-in-jenkins-x-yml",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:         "correct-pipeline-stage-is-removed",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:         "command-as-multiline-script",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:         "pipeline-timeout",
			language:     "none",
			repoName:     "js-test-repo",
			organization: "abayer",
			branch:       "really-long",
			kind:         "release",
		},
		{
			name:         "containerOptions-at-top-level-of-buildpack",
			language:     "maven-with-resource-limit",
			repoName:     "jx-demo-qs",
			organization: "abayer",
			branch:       "master",
			kind:         "release",
			useKaniko:    false,
		},
	}

	k8sObjects := []runtime.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kube.ConfigMapJenkinsDockerRegistry,
				Namespace: "jx",
			},
			Data: map[string]string{
				"docker.registry": "gcr.io",
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
			if err != nil {
				t.Fatalf("Error loading %s/jenkins-x.yml: %s", caseDir, err)
			}

			createTask := &StepCreateTaskOptions{
				Pack:         tt.language,
				DryRun:       true,
				SourceName:   "source",
				PodTemplates: assertLoadPodTemplates(t),
				GitInfo: &gits.GitRepository{
					Host:         "github.com",
					Name:         tt.repoName,
					Organisation: tt.organization,
				},
				Branch:       tt.branch,
				PipelineKind: tt.kind,
				NoKaniko:     !tt.useKaniko,
				StepOptions: opts.StepOptions{
					CommonOptions: &opts.CommonOptions{
						ServiceAccount: "tekton-bot",
					},
				},
				BuildNumber: "1",
				VersionResolver: &opts.VersionResolver{
					VersionsDir: testVersionsDir,
				},
				DefaultImage:      "maven",
				KanikoImage:       "gcr.io/kaniko-project/executor:9912ccbf8d22bbafbf971124600fbb0b13b9cbd6",
				KanikoSecretMount: "/kaniko-secret/secret.json",
				KanikoSecret:      "kaniko-secret",
				KanikoSecretKey:   "kaniko-secret",
			}
			testhelpers.ConfigureTestOptionsWithResources(createTask.CommonOptions, k8sObjects, jxObjects, gits_test.NewMockGitter(), fakeGitProvider, helm_test.NewMockHelmer(), nil)

			ns := "jx"
			// Create a single duplicate PipelineResource for the name used by the 'kaniko_entrypoint' test case to verify that the deduplication logic works correctly.
			tektonClient := tektonfake.NewSimpleClientset(tekton_helpers_test.AssertLoadPipelineResources(t, path.Join(testData, "prepopulated")))

			err = createTask.setBuildValues()
			assert.NoError(t, err)

			effectiveProjectConfig, _ := createTask.createEffectiveProjectConfig(packsDir, projectConfig, projectConfigFile, resolver, ns)
			if effectiveProjectConfig != nil {
				err = createTask.setBuildVersion(effectiveProjectConfig.PipelineConfig)
				assert.NoError(t, err)
			}

			pipelineName := tekton.PipelineResourceNameFromGitInfo(createTask.GitInfo, createTask.Branch, createTask.Context, tekton.BuildPipeline, tektonClient, ns)
			crds, err := createTask.generateTektonCRDs(effectiveProjectConfig, ns, pipelineName)
			if tt.expectingError {
				if err == nil {
					t.Fatalf("Expected an error generating CRDs")
				}
			} else {
				if err != nil {
					t.Fatalf("Error generating CRDs: %s", err)
				}

				// to update the golden files 'make test1-pkg PKG=./pkg/cmd/step/create TEST=TestGenerateTektonCRDs UPDATE_GOLDEN=1' - use with care!
				if os.Getenv("UPDATE_GOLDEN") != "" {
					err = crds.WriteToDisk(caseDir, nil)
					assert.NoError(t, err)
				}

				assertTektonCRDs(t, tt, crds, caseDir, createTask)
			}
		})
	}
}

func assertTektonCRDs(t *testing.T, testCase testCase, crds *tekton.CRDWrapper, caseDir string, createTask *StepCreateTaskOptions) {
	taskList := &pipelineapi.TaskList{}
	for _, task := range crds.Tasks() {
		taskList.Items = append(taskList.Items, *task)
	}
	resourceList := &pipelineapi.PipelineResourceList{}
	for _, resource := range crds.Resources() {
		resourceList.Items = append(resourceList.Items, *resource)
	}
	if d := cmp.Diff(tekton_helpers_test.AssertLoadSinglePipeline(t, caseDir), crds.Pipeline()); d != "" {
		t.Errorf("Generated Pipeline did not match expected: \n%s", d)
	}
	if d, _ := kmp.SafeDiff(tekton_helpers_test.AssertLoadTasks(t, caseDir), taskList, cmpopts.IgnoreFields(corev1.ResourceRequirements{}, "Requests")); d != "" {
		t.Errorf("Generated Tasks did not match expected: \n%s", d)
	}
	if d := cmp.Diff(tekton_helpers_test.AssertLoadPipelineResources(t, caseDir), resourceList); d != "" {
		t.Errorf("Generated PipelineResources did not match expected: %s", d)
	}
	if d := cmp.Diff(tekton_helpers_test.AssertLoadSinglePipelineRun(t, caseDir), crds.PipelineRun()); d != "" {
		t.Errorf("Generated PipelineRun did not match expected: %s", d)
	}
	if d := cmp.Diff(tekton_helpers_test.AssertLoadSinglePipelineStructure(t, caseDir), crds.Structure()); d != "" {
		t.Errorf("Generated PipelineStructure did not match expected: %s", d)
	}
	pa := tekton.GeneratePipelineActivity(createTask.BuildNumber, createTask.Branch, createTask.GitInfo, &prow.PullRefs{}, tekton.BuildPipeline)
	expectedActivityKey := &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:     fmt.Sprintf("%s-%s-%s-1", testCase.organization, testCase.repoName, testCase.branch),
			Pipeline: fmt.Sprintf("%s/%s/%s", testCase.organization, testCase.repoName, testCase.branch),
			Build:    "1",
			GitInfo:  createTask.GitInfo,
		},
	}
	if d := cmp.Diff(expectedActivityKey, pa); d != "" {
		t.Errorf("not match expected: %s", d)
	}
}

func assertLoadPodTemplates(t *testing.T) map[string]*corev1.Pod {
	fileName := filepath.Join("test_data", "step_create_task", "PodTemplates.yml")
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
