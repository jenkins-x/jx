package helm

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAddYamlLabels(t *testing.T) {
	t.Parallel()

	f, err := ioutil.TempDir("", "test-add-yaml-labels")
	assert.NoError(t, err)

	testData := path.Join("test_data", "set_labels")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	err = util.CopyDir(testData, f, true)
	assert.NoError(t, err)

	expectedChartName := "cheese"
	expectedChartVersion := "1.2.3"

	err = addLabelsToChartYaml(f, expectedChartName, expectedChartVersion)
	assert.NoError(t, err, "Failed to add labels to YAML")

	err = filepath.Walk(f, func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		if ext == ".yaml" {
			file := path
			svc := &corev1.Service{}
			data, err := ioutil.ReadFile(file)
			assert.NoError(t, err, "Failed to load Service YAML %s", path)
			if err == nil {
				err = yaml.Unmarshal(data, &svc)
				assert.NoError(t, err, "Failed to parse Service YAML %s", path)
				if err == nil {
					labels := svc.Labels
					assert.NotNil(t, labels, "No labels on Service %s", path)
					if labels != nil {
						key := LabelReleaseName
						actual := labels[key]
						assert.Equal(t, expectedChartName, actual, "Failed to find label %s on Service YAML %s", key, path)
						//log.Infof("Found label %s = %s for file %s\n", key, actual, path)

						key = LabelReleaseChartVersion
						actual = labels[key]
						assert.Equal(t, expectedChartVersion, actual, "Failed to find label %s on Service YAML %s", key, path)
					}
				}
			}
		}
		return nil
	})
	assert.NoError(t, err, "Failed to walk folders")
}
