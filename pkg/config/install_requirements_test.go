package config_test

import (
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/log"
	"io/ioutil"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

var (
	testDataDir = path.Join("test_data")
)

func TestRequirementsConfigMarshalExistingFile(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "test-requirements-config-")
	assert.NoError(t, err, "should create a temporary config dir")

	expectedClusterName := "my-cluster"
	expectedSecretStorage := config.SecretStorageTypeVault
	expectedDomain := "cheese.co.uk"

	file := filepath.Join(dir, config.RequirementsConfigFileName)
	requirements := config.NewRequirementsConfig()
	requirements.SecretStorage = expectedSecretStorage
	requirements.Cluster.ClusterName = expectedClusterName
	requirements.Ingress.Domain = expectedDomain
	requirements.Kaniko = true

	err = requirements.SaveConfig(file)
	assert.NoError(t, err, "failed to save file %s", file)

	requirements, fileName, err := config.LoadRequirementsConfig(dir)
	assert.NoError(t, err, "failed to load requirements file in dir %s", dir)
	assert.FileExists(t, fileName)

	assert.Equal(t, true, requirements.Kaniko, "requirements.Kaniko")
	assert.Equal(t, expectedClusterName, requirements.Cluster.ClusterName, "requirements.ClusterName")
	assert.Equal(t, expectedSecretStorage, requirements.SecretStorage, "requirements.SecretStorage")
	assert.Equal(t, expectedDomain, requirements.Ingress.Domain, "requirements.Domain")

	// lets check we can load the file from a sub dir
	subDir := filepath.Join(dir, "subdir")
	requirements, fileName, err = config.LoadRequirementsConfig(subDir)
	assert.NoError(t, err, "failed to load requirements file in subDir: %s", subDir)
	assert.FileExists(t, fileName)

	assert.Equal(t, true, requirements.Kaniko, "requirements.Kaniko")
	assert.Equal(t, expectedClusterName, requirements.Cluster.ClusterName, "requirements.ClusterName")
	assert.Equal(t, expectedSecretStorage, requirements.SecretStorage, "requirements.SecretStorage")
	assert.Equal(t, expectedDomain, requirements.Ingress.Domain, "requirements.Domain")
}

func TestRequirementsConfigMarshalExistingFileKanikoFalse(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "test-requirements-config-")
	assert.NoError(t, err, "should create a temporary config dir")

	file := filepath.Join(dir, config.RequirementsConfigFileName)
	requirements := config.NewRequirementsConfig()
	requirements.Kaniko = false

	err = requirements.SaveConfig(file)
	assert.NoError(t, err, "failed to save file %s", file)

	requirements, fileName, err := config.LoadRequirementsConfig(dir)
	assert.NoError(t, err, "failed to load requirements file in dir %s", dir)
	assert.FileExists(t, fileName)

	assert.Equal(t, false, requirements.Kaniko, "requirements.Kaniko")

}

func TestRequirementsConfigMarshalInEmptyDir(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "test-requirements-config-empty-")

	requirements, fileName, err := config.LoadRequirementsConfig(dir)
	assert.NoError(t, err, "failed to load requirements file in dir %s", dir)

	exists, err := util.FileExists(fileName)
	assert.NoError(t, err, "failed to check file exists %s", fileName)
	assert.False(t, exists, "file should not exist %s", fileName)

	assert.Equal(t, config.WebhookTypeProw, requirements.Webhook, "requirements.WebhookTypeProw")
	assert.Equal(t, false, requirements.Kaniko, "requirements.Kaniko")
	assert.Equal(t, config.SecretStorageTypeLocal, requirements.SecretStorage, "requirements.SecretStorage")
}

func TestRequirementsConfigIngressAutoDNS(t *testing.T) {
	t.Parallel()

	requirements := config.NewRequirementsConfig()

	requirements.Ingress.Domain = "1.2.3.4.nip.io"
	assert.Equal(t, true, requirements.Ingress.IsAutoDNSDomain(), "requirements.Ingress.IsAutoDNSDomain() for domain %s", requirements.Ingress.Domain)

	requirements.Ingress.Domain = "foo.bar"
	assert.Equal(t, false, requirements.Ingress.IsAutoDNSDomain(), "requirements.Ingress.IsAutoDNSDomain() for domain %s", requirements.Ingress.Domain)

	requirements.Ingress.Domain = ""
	assert.Equal(t, false, requirements.Ingress.IsAutoDNSDomain(), "requirements.Ingress.IsAutoDNSDomain() for domain %s", requirements.Ingress.Domain)
}

func Test_env_repository_visibility(t *testing.T) {
	t.Parallel()

	var gitPublicTests = []struct {
		yamlFile          string
		expectedGitPublic bool
	}{
		{"git_public_nil_git_private_true.yaml", false},
		{"git_public_nil_git_private_false.yaml", true},
		{"git_public_false_git_private_nil.yaml", false},
		{"git_public_true_git_private_nil.yaml", true},
	}

	for _, testCase := range gitPublicTests {
		t.Run(testCase.yamlFile, func(t *testing.T) {
			content, err := ioutil.ReadFile(path.Join(testDataDir, testCase.yamlFile))
			assert.NoError(t, err)

			config := config.NewRequirementsConfig()

			_ = log.CaptureOutput(func() {
				err = yaml.Unmarshal(content, config)
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedGitPublic, config.Cluster.EnvironmentGitPublic, "unexpected value for repository visibility")
			})
		})
	}
}

func Test_EnvironmentGitPublic_and_EnvironmentGitPrivate_specified_together_return_error(t *testing.T) {
	content, err := ioutil.ReadFile(path.Join(testDataDir, "git_public_true_git_private_true.yaml"))
	assert.NoError(t, err)

	config := config.NewRequirementsConfig()
	err = yaml.Unmarshal(content, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only EnvironmentGitPublic should be used")
}
