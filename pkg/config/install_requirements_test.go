package config_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestRequirementsConfigMarshalExistingFile(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "test-requirements-config-")
	assert.NoError(t, err, "should create a temporary config dir")

	expectedClusterName := "my-cluster"
	expectedSecretStorage := config.SecretStorageTypeVault

	file := filepath.Join(dir, config.RequirementsConfigFileName)
	requirements := config.NewRequirementsConfig()
	requirements.SecretStorage = expectedSecretStorage
	requirements.ClusterName = expectedClusterName

	err = requirements.SaveConfig(file)
	assert.NoError(t, err, "failed to save file %s", file)

	requirements, fileName, err := config.LoadRequirementsConfig(dir)
	assert.NoError(t, err, "failed to load requirements file in dir %s", dir)
	assert.FileExists(t, fileName)

	assert.Equal(t, true, requirements.Kaniko, "requirements.Kaniko")
	assert.Equal(t, expectedClusterName, requirements.ClusterName, "requirements.ClusterName")
	assert.Equal(t, expectedSecretStorage, requirements.SecretStorage, "requirements.SecretStorage")

	// lets check we can load the file from a sub dir
	subDir := filepath.Join(dir, "subdir")
	requirements, fileName, err = config.LoadRequirementsConfig(subDir)
	assert.NoError(t, err, "failed to load requirements file in subDir: %s", subDir)
	assert.FileExists(t, fileName)

	assert.Equal(t, true, requirements.Kaniko, "requirements.Kaniko")
	assert.Equal(t, expectedClusterName, requirements.ClusterName, "requirements.ClusterName")
	assert.Equal(t, expectedSecretStorage, requirements.SecretStorage, "requirements.SecretStorage")
}

func TestRequirementsConfigMarshalInEmptyDir(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "test-requirements-config-empty-")

	requirements, fileName, err := config.LoadRequirementsConfig(dir)
	assert.NoError(t, err, "failed to load requirements file in dir %s", dir)

	exists, err := util.FileExists(fileName)
	assert.NoError(t, err, "failed to check file exists %s", fileName)
	assert.False(t, exists, "file should not exist %s", fileName)

	assert.Equal(t, true, requirements.Kaniko, "requirements.Kaniko")
	assert.Equal(t, config.SecretStorageTypeLocal, requirements.SecretStorage, "requirements.SecretStorage")
}
