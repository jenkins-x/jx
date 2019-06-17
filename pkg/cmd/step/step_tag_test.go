package step_test

import (
	"github.com/jenkins-x/jx/pkg/cmd/step"
	"path/filepath"
	"strings"
	"testing"

	"io/ioutil"
	"os"
	"path"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/helm/pkg/chartutil"
)

func TestStepTagCharts(t *testing.T) {
	t.Parallel()
	f, err := ioutil.TempDir("", "test-step-tag-charts")
	assert.NoError(t, err)

	testData := path.Join("test_data", "step_tag_project")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	err = util.CopyDir(testData, f, true)
	assert.NoError(t, err)

	expectedVersion := "1.2.3"
	expectedImageName := "docker.io/jenkinsxio/awesome"

	chartsDir := filepath.Join(f, "charts", "mydemo")
	chartFile := filepath.Join(chartsDir, "Chart.yaml")
	valuesFile := filepath.Join(chartsDir, "values.yaml")

	o := step.StepTagOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	//o.Out = tests.Output()
	o.Flags.ChartsDir = chartsDir
	o.Flags.Version = expectedVersion
	o.Flags.ChartValueRepository = expectedImageName
	o.SetGit(&gits.GitFake{})
	err = o.Run()
	assert.NoError(t, err)

	// root file
	chart, err := chartutil.LoadChartfile(chartFile)
	assert.NoError(t, err, "failed to load file %s", chartFile)

	assert.Equal(t, expectedVersion, chart.Version, "replaced chart version")
	assert.Equal(t, expectedVersion, chart.AppVersion, "replaced chart appVersion")

	data, err := ioutil.ReadFile(valuesFile)
	assert.NoError(t, err, "failed to load file %s", valuesFile)
	lines := strings.Split(string(data), "\n")

	foundRepo := false
	foundVersion := false
	anotherImage := false
	for _, line := range lines {
		if strings.HasPrefix(line, "anotherImage:") {
			anotherImage = true
		}
		if strings.HasPrefix(line, step.ValuesYamlRepositoryPrefix) {
			value := strings.TrimSpace(strings.TrimPrefix(line, step.ValuesYamlRepositoryPrefix))
			if anotherImage {
				assert.Equal(t, "anotherImageRepoValue", value, "values.yaml anotherImage.repository: attribute")
			} else {
				foundRepo = true
				assert.Equal(t, expectedImageName, value, "values.yaml repository: attribute")
			}
		} else if strings.HasPrefix(line, step.ValuesYamlTagPrefix) {
			value := strings.TrimSpace(strings.TrimPrefix(line, step.ValuesYamlTagPrefix))
			if anotherImage {
				assert.Equal(t, "anotherImageTagValue", value, "values.yaml anotherImage.tag: attribute")
			} else {
				foundVersion = true
				assert.Equal(t, expectedVersion, value, "values.yaml tag: attribute")
			}
		}
	}

	assert.True(t, foundRepo, "Failed to find tag '%s' in file %s", step.ValuesYamlRepositoryPrefix, valuesFile)
	assert.True(t, foundVersion, "Failed to find tag '%s' in file %s", step.ValuesYamlTagPrefix, valuesFile)
}
