package helmfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	helmfile2 "github.com/jenkins-x/jx/pkg/helmfile"

	"github.com/jenkins-x/jx/pkg/cmd/create/options"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestDedupeRepositories(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-applications-config")
	assert.NoError(t, err, "should create a temporary config dir")

	o := &CreateHelmfileOptions{
		outputDir:     tempDir,
		dir:           "test_data",
		CreateOptions: *getCreateOptions(),
	}
	err = o.Run()
	assert.NoError(t, err)

	h, err := loadHelmfile(tempDir)
	assert.NoError(t, err)

	// assert there are 3 repos and not 4 as one of them in the jx-applications.yaml is a duplicate
	assert.Equal(t, 3, len(h.Repositories))

}

func TestExtraAppValues(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-applications-config")
	assert.NoError(t, err, "should create a temporary config dir")

	o := &CreateHelmfileOptions{
		outputDir:     tempDir,
		dir:           path.Join("test_data", "extra-values"),
		CreateOptions: *getCreateOptions(),
	}
	err = o.Run()
	assert.NoError(t, err)

	h, err := loadHelmfile(tempDir)
	assert.NoError(t, err)

	// assert we added the local values.yaml for the velero app
	assert.Equal(t, "velero/values.yaml", h.Releases[0].Values[0])

}

func TestExtraFlagValues(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-applications-config")
	assert.NoError(t, err, "should create a temporary config dir")

	o := &CreateHelmfileOptions{
		outputDir:     tempDir,
		dir:           path.Join("test_data"),
		valueFiles:    []string{"foo/bar.yaml"},
		CreateOptions: *getCreateOptions(),
	}
	err = o.Run()
	assert.NoError(t, err)

	h, err := loadHelmfile(tempDir)
	assert.NoError(t, err)

	// assert we added the values file passed in as a CLI flag
	assert.Equal(t, "foo/bar.yaml", h.Releases[0].Values[0])

}

func loadHelmfile(dir string) (*helmfile2.HelmState, error) {

	fileName := helmfile
	if dir != "" {
		fileName = filepath.Join(dir, helmfile)
	}

	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		return nil, errors.Errorf("no %s found in directory %s", fileName, dir)
	}

	config := &helmfile2.HelmState{}

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return config, fmt.Errorf("Failed to load file %s due to %s", fileName, err)
	}
	validationErrors, err := util.ValidateYaml(config, data)
	if err != nil {
		return config, fmt.Errorf("failed to validate YAML file %s due to %s", fileName, err)
	}
	if len(validationErrors) > 0 {
		return config, fmt.Errorf("Validation failures in YAML file %s:\n%s", fileName, strings.Join(validationErrors, "\n"))
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return config, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
	}

	return config, err
}

func getCreateOptions() *options.CreateOptions {

	helmer := helm_test.NewMockHelmer()
	co := &opts.CommonOptions{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}
	co.SetHelm(helmer)
	return &options.CreateOptions{
		CommonOptions: co,
	}
}
