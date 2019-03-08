package helm

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAddYamlLabels(t *testing.T) {
	t.Parallel()

	baseDir, err := ioutil.TempDir("", "test-add-yaml-labels")
	assert.NoError(t, err)

	testData := path.Join("test_data", "set_labels")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	outDir := path.Join(baseDir, "output")
	hooksDir := path.Join(baseDir, "hooks")
	err = util.CopyDir(testData, outDir, true)
	assert.NoError(t, err)

	expectedChartName := "mychart"
	expectedChartRelease := "cheese"
	expectedChartVersion := "1.2.3"
	expectedNamespace := "jx"

	chartMetadata := &chart.Metadata{
		Name:    expectedChartName,
		Version: expectedChartVersion,
	}

	helmHooks, err := addLabelsToChartYaml(outDir, hooksDir, expectedChartName, expectedChartRelease, expectedChartVersion, chartMetadata, expectedNamespace)
	assert.NoError(t, err, "Failed to add labels to YAML")

	err = filepath.Walk(outDir, func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		if ext == ".yaml" {
			file := path
			svc := &corev1.Service{}
			data, err := ioutil.ReadFile(file)
			assert.NoError(t, err, "Failed to load YAML %s", path)
			if err == nil {
				err = yaml.Unmarshal(data, &svc)
				assert.NoError(t, err, "Failed to parse YAML %s", path)
				if err == nil {
					labels := svc.Labels
					assert.NotNil(t, labels, "No labels on path %s", path)
					if labels != nil {
						assertLabelValue(t, expectedChartRelease, labels, LabelReleaseName, path)
						assertLabelValue(t, expectedChartVersion, labels, LabelReleaseChartVersion, path)

						_, fileName := filepath.Split(file)
						if fileName == "clusterrole.yaml" {
							assertLabelValue(t, expectedNamespace, labels, LabelNamespace, path)
						} else {
							assertNoLabelValue(t, labels, LabelNamespace, path)
						}
					}
				}
			}
		}
		return nil
	})

	assert.FileExists(t, filepath.Join(hooksDir, "post-install-job.yaml"), "Should have moved this YAML into the hooks dir!")

	if assert.Equal(t, 1, len(helmHooks), "number of helm hooks") {
		hook := helmHooks[0]
		if assert.NotNil(t, hook, "helm hook") {
			assert.Equal(t, []string{"post-install", "post-upgrade"}, hook.Hooks, "hooks")
			assert.Equal(t, []string{"hook-succeeded"}, hook.HookDeletePolicies, "hook delete policies")
			assert.FileExists(t, hook.File, "should have a helm hook template file")
		}
	}
	assert.NoError(t, err, "Failed to walk folders")
}

func assertLabelValue(t *testing.T, expectedValue string, labels map[string]string, key string, path string) bool {
	require.NotNil(t, labels, "labels were nil for path %s", path)
	actual := labels[key]
	return assert.Equal(t, expectedValue, actual, "Failed to find label %s on AML %s", key, path)
}

func assertNoLabelValue(t *testing.T, labels map[string]string, key string, path string) bool {
	if labels != nil {
		actual := labels[key]
		return assert.Equal(t, "", actual, "Should not have label %s on YAML %s", key, path)
	}
	return true
}

func TestSplitObjectsInFiles(t *testing.T) {
	t.Parallel()

	dataDir := filepath.Join("test_data", "multi_objects")
	testDir, err := ioutil.TempDir("", "test_multi_objects")
	assert.NoError(t, err, "should crate a temp dir for tests")
	err = util.CopyDir(dataDir, testDir, true)
	assert.NoError(t, err, "shold copy the test data into a temporary folder")
	defer os.RemoveAll(testDir)

	tests := map[string]struct {
		file    string
		want    int
		wantErr bool
	}{
		"single object": {
			file:    filepath.Join(testDir, "single_object.yaml"),
			want:    1,
			wantErr: false,
		},
		"single object with separator": {
			file:    filepath.Join(testDir, "single_object_separator.yaml"),
			want:    1,
			wantErr: false,
		},
		"multiple objects": {
			file:    filepath.Join(testDir, "objects.yaml"),
			want:    2,
			wantErr: false,
		},
		"multiple objects with separator": {
			file:    filepath.Join(testDir, "objects_separator.yaml"),
			want:    2,
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			parts, err := splitObjectsInFiles(tc.file)
			if tc.wantErr {
				assert.Error(t, err, "should fail")
			} else {
				assert.NoError(t, err, "should not fail")
			}
			assert.Equal(t, tc.want, len(parts))
		})
	}
}
