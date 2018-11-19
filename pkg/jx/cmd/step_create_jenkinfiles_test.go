package cmd_test

import (
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepCreateJenkinsfile(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	t.Parallel()
	tempDir, err := ioutil.TempDir("", "test-step-create-jenkinsfile")
	assert.NoError(t, err)

	testData := path.Join("test_data", "step_create_jenkinsfile")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	files, err := ioutil.ReadDir(testData)
	assert.NoError(t, err)

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			srcDir := filepath.Join(testData, name)
			outDir := filepath.Join(tempDir, name)
			testStepCreateJenkinsfile(t, outDir, name, srcDir)
		}
	}
}

func testStepCreateJenkinsfile(t *testing.T, outDir string, testcase string, srcDir string) error {
	configFile := path.Join(srcDir, jenkinsfile.PipelineConfigFileName)
	templateFile := path.Join(srcDir, jenkinsfile.PipelineTemplateFileName)
	expectedFile := path.Join(srcDir, "Jenkinsfile")
	actualFile := path.Join(outDir, "Jenkinsfile")

	assert.FileExists(t, configFile)
	assert.FileExists(t, templateFile)
	assert.FileExists(t, expectedFile)

	o := &cmd.StepCreateJenkinsfileOptions{}

	err := o.GenerateJenkinsfile(&jenkinsfile.CreateJenkinsfileArguments{
		ConfigFile:   configFile,
		TemplateFile: templateFile,
		OutputFile:   actualFile,
	})
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

	config := &jenkinsfile.PipelineConfig{
		Agent: jenkinsfile.PipelineAgent{
			Label: "jenkins-maven",
		},
		Pipelines: jenkinsfile.Pipelines{
			Release: &jenkinsfile.PipelineLifecycles{
				Setup: &jenkinsfile.PipelineLifecycle{
					Steps: []*jenkinsfile.PipelineStep{
						{
							Container: "maven",
							Steps: []*jenkinsfile.PipelineStep{
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
	pipelineFile := path.Join("test_data", "step_create_jenkinsfile", jenkinsfile.PipelineConfigFileName)
	assert.FileExists(t, pipelineFile)

	config, err := jenkinsfile.LoadPipelineConfig(pipelineFile)
	require.NoError(t, err, "failed to load pipeline config %s", pipelineFile)

	assert.Equal(t, "jenkins-maven", config.Agent.Label, "Agent.Label")
	assert.NotNil(t, config.Pipelines.Release, "Pipelines.Release")
}

func TestParseLongerPipelineConfig(t *testing.T) {
	pipelineFile := path.Join("test_data", "step_create_jenkinsfile", "simple", jenkinsfile.PipelineConfigFileName)
	assert.FileExists(t, pipelineFile)

	config, err := jenkinsfile.LoadPipelineConfig(pipelineFile)
	require.NoError(t, err, "failed to load pipeline config %s", pipelineFile)

	assert.Equal(t, "jenkins-maven", config.Agent.Label, "Agent.Label")
	assert.NotNil(t, config.Pipelines.PullRequest, "Pipelines.PullRequest")
	assert.NotNil(t, config.Pipelines.Release, "Pipelines.Release")
}
