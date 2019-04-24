package cmd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/syntax/syntax.jenkins.io/v1alpha1"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestCreateJenkinsfile(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	t.Parallel()
	tempDir, err := ioutil.TempDir("", "test-step-buildpack-apply")
	assert.NoError(t, err)

	testData := path.Join("test_data", "step_buildpack_apply")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	files, err := ioutil.ReadDir(testData)
	assert.NoError(t, err)

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			srcDir := filepath.Join(testData, name)
			outDir := filepath.Join(tempDir, name)
			testCreateJenkinsfile(t, outDir, name, srcDir)
		}
	}
}

func testCreateJenkinsfile(t *testing.T, outDir string, testcase string, srcDir string) error {
	configFile := path.Join(srcDir, v1alpha1.PipelineConfigFileName)
	templateFile := path.Join(srcDir, v1alpha1.PipelineTemplateFileName)
	expectedFile := path.Join(srcDir, "Jenkinsfile")
	actualFile := path.Join(outDir, "Jenkinsfile")

	assert.FileExists(t, configFile)
	assert.FileExists(t, templateFile)
	assert.FileExists(t, expectedFile)

	resolver := func(importFile *jenkinsfile.ImportFile) (string, error) {
		dirPath := []string{srcDir, "import_dir", importFile.Import}
		// lets handle cross platform paths in `importFile.File`
		path := append(dirPath, strings.Split(importFile.File, "/")...)
		return filepath.Join(path...), nil
	}

	arguments := &v1alpha1.CreateJenkinsfileArguments{
		ConfigFile:   configFile,
		TemplateFile: templateFile,
		OutputFile:   actualFile,
	}
	if testcase == "prow" || strings.HasPrefix(testcase, "prow_") {
		arguments.JenkinsfileRunner = true
		arguments.ClearContainerNames = true
	}

	err := arguments.GenerateJenkinsfile(resolver)
	assert.NoError(t, err, "Failed with %s", err)
	assert.FileExists(t, expectedFile)
	if err == nil {
		err = tests.AssertEqualFileText(t, expectedFile, actualFile)
		if err != nil {
			data, err := ioutil.ReadFile(actualFile)
			if err == nil {
				t.Logf("Generated file\n%s\n", string(data))
			}
			return err
		}
	}
	return err
}

func TestSavePipelineConfig(t *testing.T) {
	t.Parallel()
	tempDir, err := ioutil.TempDir("", "test-step-save-pipeline-config")
	assert.NoError(t, err)

	file := filepath.Join(tempDir, "pipeline.yaml")

	config := &v1alpha1.PipelineConfig{
		Agent: v1alpha1.PipelineAgent{
			Label: "jenkins-maven",
		},
		Pipelines: v1alpha1.Pipelines{
			Release: &v1alpha1.PipelineLifecycles{
				Setup: &v1alpha1.PipelineLifecycle{
					Steps: []*v1alpha1.PipelineStep{
						{
							Container: "maven",
							Steps: []*v1alpha1.PipelineStep{
								{
									Command: "mvn deploy",
								},
								{
									Groovy: `dir("foo") { 
  sh "echo hello"
}`,
								},
							},
						},
					},
				},
			},
		},
	}

	err = config.SaveConfig(file)
	require.NoError(t, err, "failed to save pipeline config %s", file)

	t.Logf("saved pipeline file to %s\n", file)
}

func TestParsePipelineConfig(t *testing.T) {
	pipelineFile := path.Join("test_data", "step_buildpack_apply", v1alpha1.PipelineConfigFileName)
	assert.FileExists(t, pipelineFile)

	config, err := v1alpha1.LoadPipelineConfig(pipelineFile, dummyImportFileResolver, false, false)
	require.NoError(t, err, "failed to load pipeline config %s", pipelineFile)

	assert.Equal(t, "jenkins-maven", config.Agent.Label, "Agent.Label")
	assert.NotNil(t, config.Pipelines.Release, "Pipelines.Release")
}

func TestParseLongerPipelineConfig(t *testing.T) {
	pipelineFile := path.Join("test_data", "step_buildpack_apply", "simple", v1alpha1.PipelineConfigFileName)
	assert.FileExists(t, pipelineFile)

	config, err := v1alpha1.LoadPipelineConfig(pipelineFile, dummyImportFileResolver, false, false)
	require.NoError(t, err, "failed to load pipeline config %s", pipelineFile)

	assert.Equal(t, "jenkins-maven", config.Agent.Label, "Agent.Label")
	assert.NotNil(t, config.Pipelines.PullRequest, "Pipelines.PullRequest")
	assert.NotNil(t, config.Pipelines.Release, "Pipelines.Release")
}

func dummyImportFileResolver(importFile *jenkinsfile.ImportFile) (string, error) {
	return importFile.File, nil

}
