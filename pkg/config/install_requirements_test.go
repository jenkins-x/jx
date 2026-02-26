// +build unit

package config_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/cloud/gke"
	"github.com/stretchr/testify/require"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cloud"
	"github.com/jenkins-x/jx/v2/pkg/config"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/jenkins-x/jx/v2/pkg/util"
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

	requirements, fileName, err := config.LoadRequirementsConfig(dir, config.DefaultFailOnValidationError)
	assert.NoError(t, err, "failed to load requirements file in dir %s", dir)
	assert.FileExists(t, fileName)

	assert.Equal(t, true, requirements.Kaniko, "requirements.Kaniko")
	assert.Equal(t, expectedClusterName, requirements.Cluster.ClusterName, "requirements.ClusterName")
	assert.Equal(t, expectedSecretStorage, requirements.SecretStorage, "requirements.SecretStorage")
	assert.Equal(t, expectedDomain, requirements.Ingress.Domain, "requirements.Domain")

	// lets check we can load the file from a sub dir
	subDir := filepath.Join(dir, "subdir")
	requirements, fileName, err = config.LoadRequirementsConfig(subDir, config.DefaultFailOnValidationError)
	assert.NoError(t, err, "failed to load requirements file in subDir: %s", subDir)
	assert.FileExists(t, fileName)

	assert.Equal(t, true, requirements.Kaniko, "requirements.Kaniko")
	assert.Equal(t, expectedClusterName, requirements.Cluster.ClusterName, "requirements.ClusterName")
	assert.Equal(t, expectedSecretStorage, requirements.SecretStorage, "requirements.SecretStorage")
	assert.Equal(t, expectedDomain, requirements.Ingress.Domain, "requirements.Domain")
}

func Test_OverrideRequirementsFromEnvironment_does_not_initialise_nil_structs(t *testing.T) {
	requirements, fileName, err := config.LoadRequirementsConfig(testDataDir, config.DefaultFailOnValidationError)
	assert.NoError(t, err, "failed to load requirements file in dir %s", testDataDir)
	assert.FileExists(t, fileName)

	requirements.OverrideRequirementsFromEnvironment(func() gke.GClouder {
		return nil
	})

	tempDir, err := ioutil.TempDir("", "test-requirements-config")
	assert.NoError(t, err, "should create a temporary config dir")
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	err = requirements.SaveConfig(filepath.Join(tempDir, config.RequirementsConfigFileName))
	assert.NoError(t, err, "failed to save requirements file in dir %s", tempDir)

	overrideRequirements, fileName, err := config.LoadRequirementsConfig(tempDir, config.DefaultFailOnValidationError)
	assert.NoError(t, err, "failed to load requirements file in dir %s", testDataDir)
	assert.FileExists(t, fileName)

	assert.Nil(t, overrideRequirements.BuildPacks, "nil values should not be populated")
}

func Test_OverrideRequirementsFromEnvironment_populate_requirements_from_environment_variables(t *testing.T) {
	var overrideTests = []struct {
		envKey               string
		envValue             string
		expectedRequirements config.RequirementsConfig
	}{
		// RequirementsConfig
		{config.RequirementSecretStorageType, "vault", config.RequirementsConfig{SecretStorage: "vault"}},
		{config.RequirementKaniko, "true", config.RequirementsConfig{Kaniko: true}},
		{config.RequirementKaniko, "false", config.RequirementsConfig{Kaniko: false}},
		{config.RequirementKaniko, "", config.RequirementsConfig{Kaniko: false}},
		{config.RequirementRepository, "bucketrepo", config.RequirementsConfig{Repository: "bucketrepo"}},
		{config.RequirementWebhook, "prow", config.RequirementsConfig{Webhook: "prow"}},
		{config.RequirementGitAppEnabled, "true", config.RequirementsConfig{GithubApp: &config.GithubAppConfig{Enabled: true}}},
		{config.RequirementGitAppEnabled, "false", config.RequirementsConfig{GithubApp: &config.GithubAppConfig{Enabled: false}}},
		{config.RequirementGitAppURL, "https://my-github-app", config.RequirementsConfig{GithubApp: &config.GithubAppConfig{URL: "https://my-github-app"}}},

		// ClusterConfig
		{config.RequirementClusterName, "my-cluster", config.RequirementsConfig{Cluster: config.ClusterConfig{ClusterName: "my-cluster"}}},
		{config.RequirementProject, "my-project", config.RequirementsConfig{Cluster: config.ClusterConfig{ProjectID: "my-project"}}},
		{config.RequirementZone, "my-zone", config.RequirementsConfig{Cluster: config.ClusterConfig{Zone: "my-zone"}}},
		{config.RequirementChartRepository, "my-chart-museum", config.RequirementsConfig{Cluster: config.ClusterConfig{ChartRepository: "my-chart-museum"}}},
		{config.RequirementRegistry, "my-registry", config.RequirementsConfig{Cluster: config.ClusterConfig{Registry: "my-registry"}}},
		{config.RequirementEnvGitOwner, "john-doe", config.RequirementsConfig{Cluster: config.ClusterConfig{EnvironmentGitOwner: "john-doe"}}},
		{config.RequirementKanikoServiceAccountName, "kaniko-sa", config.RequirementsConfig{Cluster: config.ClusterConfig{KanikoSAName: "kaniko-sa"}}},
		{config.RequirementEnvGitPublic, "true", config.RequirementsConfig{Cluster: config.ClusterConfig{EnvironmentGitPublic: true}}},
		{config.RequirementEnvGitPublic, "false", config.RequirementsConfig{Cluster: config.ClusterConfig{EnvironmentGitPublic: false}}},
		{config.RequirementEnvGitPublic, "", config.RequirementsConfig{Cluster: config.ClusterConfig{EnvironmentGitPublic: false}}},
		{config.RequirementGitPublic, "true", config.RequirementsConfig{Cluster: config.ClusterConfig{GitPublic: true}}},
		{config.RequirementGitPublic, "false", config.RequirementsConfig{Cluster: config.ClusterConfig{GitPublic: false}}},
		{config.RequirementGitPublic, "", config.RequirementsConfig{Cluster: config.ClusterConfig{GitPublic: false}}},
		{config.RequirementExternalDNSServiceAccountName, "externaldns-sa", config.RequirementsConfig{Cluster: config.ClusterConfig{ExternalDNSSAName: "externaldns-sa"}}},

		// VaultConfig
		{config.RequirementVaultName, "my-vault", config.RequirementsConfig{Vault: config.VaultConfig{Name: "my-vault"}}},
		{config.RequirementVaultServiceAccountName, "my-vault-sa", config.RequirementsConfig{Vault: config.VaultConfig{ServiceAccount: "my-vault-sa"}}},
		{config.RequirementVaultKeyringName, "my-keyring", config.RequirementsConfig{Vault: config.VaultConfig{Keyring: "my-keyring"}}},
		{config.RequirementVaultKeyName, "my-key", config.RequirementsConfig{Vault: config.VaultConfig{Key: "my-key"}}},
		{config.RequirementVaultBucketName, "my-bucket", config.RequirementsConfig{Vault: config.VaultConfig{Bucket: "my-bucket"}}},
		{config.RequirementVaultRecreateBucket, "true", config.RequirementsConfig{Vault: config.VaultConfig{RecreateBucket: true}}},
		{config.RequirementVaultRecreateBucket, "false", config.RequirementsConfig{Vault: config.VaultConfig{RecreateBucket: false}}},
		{config.RequirementVaultRecreateBucket, "", config.RequirementsConfig{Vault: config.VaultConfig{RecreateBucket: false}}},
		{config.RequirementVaultDisableURLDiscovery, "true", config.RequirementsConfig{Vault: config.VaultConfig{DisableURLDiscovery: true}}},
		{config.RequirementVaultDisableURLDiscovery, "false", config.RequirementsConfig{Vault: config.VaultConfig{DisableURLDiscovery: false}}},
		{config.RequirementVaultDisableURLDiscovery, "", config.RequirementsConfig{Vault: config.VaultConfig{DisableURLDiscovery: false}}},

		// VeleroConfig
		{config.RequirementVeleroServiceAccountName, "my-velero-sa", config.RequirementsConfig{Velero: config.VeleroConfig{ServiceAccount: "my-velero-sa"}}},
		{config.RequirementVeleroTTL, "60", config.RequirementsConfig{Velero: config.VeleroConfig{TimeToLive: "60"}}},
		{config.RequirementVeleroSchedule, "0 * * * *", config.RequirementsConfig{Velero: config.VeleroConfig{Schedule: "0 * * * *"}}},

		// IngressConfig
		{config.RequirementDomainIssuerURL, "my-issuer-url", config.RequirementsConfig{Ingress: config.IngressConfig{DomainIssuerURL: "my-issuer-url"}}},

		// Storage
		{config.RequirementStorageBackupEnabled, "true", config.RequirementsConfig{Storage: config.StorageConfig{Backup: config.StorageEntryConfig{Enabled: true}}}},
		{config.RequirementStorageBackupEnabled, "false", config.RequirementsConfig{Storage: config.StorageConfig{Backup: config.StorageEntryConfig{Enabled: false}}}},
		{config.RequirementStorageBackupEnabled, "", config.RequirementsConfig{Storage: config.StorageConfig{Backup: config.StorageEntryConfig{Enabled: false}}}},
		{config.RequirementStorageBackupURL, "gs://my-backup", config.RequirementsConfig{Storage: config.StorageConfig{Backup: config.StorageEntryConfig{URL: "gs://my-backup"}}}},

		{config.RequirementStorageLogsEnabled, "true", config.RequirementsConfig{Storage: config.StorageConfig{Logs: config.StorageEntryConfig{Enabled: true}}}},
		{config.RequirementStorageLogsEnabled, "false", config.RequirementsConfig{Storage: config.StorageConfig{Logs: config.StorageEntryConfig{Enabled: false}}}},
		{config.RequirementStorageLogsEnabled, "", config.RequirementsConfig{Storage: config.StorageConfig{Logs: config.StorageEntryConfig{Enabled: false}}}},
		{config.RequirementStorageLogsURL, "gs://my-logs", config.RequirementsConfig{Storage: config.StorageConfig{Logs: config.StorageEntryConfig{URL: "gs://my-logs"}}}},

		{config.RequirementStorageReportsEnabled, "true", config.RequirementsConfig{Storage: config.StorageConfig{Reports: config.StorageEntryConfig{Enabled: true}}}},
		{config.RequirementStorageReportsEnabled, "false", config.RequirementsConfig{Storage: config.StorageConfig{Reports: config.StorageEntryConfig{Enabled: false}}}},
		{config.RequirementStorageReportsEnabled, "", config.RequirementsConfig{Storage: config.StorageConfig{Reports: config.StorageEntryConfig{Enabled: false}}}},
		{config.RequirementStorageReportsURL, "gs://my-reports", config.RequirementsConfig{Storage: config.StorageConfig{Reports: config.StorageEntryConfig{URL: "gs://my-reports"}}}},

		{config.RequirementStorageRepositoryEnabled, "true", config.RequirementsConfig{Storage: config.StorageConfig{Repository: config.StorageEntryConfig{Enabled: true}}}},
		{config.RequirementStorageRepositoryEnabled, "false", config.RequirementsConfig{Storage: config.StorageConfig{Repository: config.StorageEntryConfig{Enabled: false}}}},
		{config.RequirementStorageRepositoryEnabled, "", config.RequirementsConfig{Storage: config.StorageConfig{Repository: config.StorageEntryConfig{Enabled: false}}}},
		{config.RequirementStorageRepositoryURL, "gs://my-repo", config.RequirementsConfig{Storage: config.StorageConfig{Repository: config.StorageEntryConfig{URL: "gs://my-repo"}}}},

		// GKEConfig
		{config.RequirementGkeProjectNumber, "my-gke-project", config.RequirementsConfig{Cluster: config.ClusterConfig{GKEConfig: &config.GKEConfig{ProjectNumber: "my-gke-project"}}}},

		// VersionStreamConfig
		{config.RequirementVersionsGitRef, "v1.0.0", config.RequirementsConfig{VersionStream: config.VersionStreamConfig{Ref: "v1.0.0"}}},
	}

	for _, overrideTest := range overrideTests {
		origEnvValue, origValueSet := os.LookupEnv(overrideTest.envKey)
		err := os.Setenv(overrideTest.envKey, overrideTest.envValue)
		assert.NoError(t, err)
		resetEnvVariable := func() {
			var err error
			if origValueSet {
				err = os.Setenv(overrideTest.envKey, origEnvValue)
			} else {
				err = os.Unsetenv(overrideTest.envKey)
			}
			if err != nil {
				log.Logger().Warnf("error resetting environment after test: %v", err)
			}
		}

		t.Run(overrideTest.envKey, func(t *testing.T) {
			actualRequirements := config.RequirementsConfig{}
			actualRequirements.OverrideRequirementsFromEnvironment(func() gke.GClouder {
				return nil
			})

			assert.Equal(t, overrideTest.expectedRequirements, actualRequirements)
		})

		resetEnvVariable()
	}
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

	requirements, fileName, err := config.LoadRequirementsConfig(dir, config.DefaultFailOnValidationError)
	assert.NoError(t, err, "failed to load requirements file in dir %s", dir)
	assert.FileExists(t, fileName)

	assert.Equal(t, false, requirements.Kaniko, "requirements.Kaniko")

}

func TestRequirementsConfigMarshalInEmptyDir(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "test-requirements-config-empty-")

	requirements, fileName, err := config.LoadRequirementsConfig(dir, config.DefaultFailOnValidationError)
	assert.Error(t, err)
	assert.Empty(t, fileName)
	assert.Nil(t, requirements)
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

func Test_unmarshalling_requirements_config_with_build_pack_configuration_succeeds(t *testing.T) {
	t.Parallel()

	requirements := config.NewRequirementsConfig()

	content, err := ioutil.ReadFile(path.Join(testDataDir, "build_pack_library.yaml"))

	err = yaml.Unmarshal(content, requirements)
	assert.NoError(t, err)
	assert.Equal(t, "Test name", requirements.BuildPacks.BuildPackLibrary.Name, "requirements.buildPacks.BuildPackLibrary.name is not equivalent to test name")
	assert.Equal(t, "github.com", requirements.BuildPacks.BuildPackLibrary.GitURL, "requirements.buildPacks.BuildPackLibrary.gitURL is not equivalent to git url ")
	assert.Equal(t, "master", requirements.BuildPacks.BuildPackLibrary.GitRef, "requirements.buildPacks.BuildPackLibrary.gitRef is not equivalent git Ref")
}

func Test_marshalling_empty_requirements_config_creates_no_build_pack_configuration(t *testing.T) {
	t.Parallel()

	requirements := config.NewRequirementsConfig()
	data, err := yaml.Marshal(requirements)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "buildPacks")

	err = yaml.Unmarshal(data, requirements)
	assert.NoError(t, err)
	assert.Nil(t, requirements.BuildPacks)
}

func Test_marshalling_vault_config(t *testing.T) {
	t.Parallel()

	requirements := config.NewRequirementsConfig()
	requirements.Vault = config.VaultConfig{
		Name:                   "myVault",
		URL:                    "http://myvault",
		ServiceAccount:         "vault-sa",
		Namespace:              "jx",
		KubernetesAuthPath:     "kubernetes",
		SecretEngineMountPoint: "secret",
	}
	data, err := yaml.Marshal(requirements)
	assert.NoError(t, err)

	assert.Contains(t, string(data), "name: myVault")
	assert.Contains(t, string(data), "url: http://myvault")
	assert.Contains(t, string(data), "serviceAccount: vault-sa")
	assert.Contains(t, string(data), "namespace: jx")
	assert.Contains(t, string(data), "kubernetesAuthPath: kubernetes")
	assert.Contains(t, string(data), "secretEngineMountPoint: secret")
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

func TestMergeSave(t *testing.T) {
	t.Parallel()
	type TestSpec struct {
		Name           string
		Original       *config.RequirementsConfig
		Changed        *config.RequirementsConfig
		ValidationFunc func(orig *config.RequirementsConfig, ch *config.RequirementsConfig)
	}

	testCases := []TestSpec{
		{
			Name: "Merge Cluster Config Test",
			Original: &config.RequirementsConfig{
				Cluster: config.ClusterConfig{
					EnvironmentGitOwner:  "owner",
					EnvironmentGitPublic: false,
					GitPublic:            false,
					Provider:             cloud.GKE,
					Namespace:            "jx",
					ProjectID:            "project-id",
					ClusterName:          "cluster-name",
					Region:               "region",
					GitKind:              gits.KindGitHub,
					GitServer:            gits.KindGitHub,
				},
			},
			Changed: &config.RequirementsConfig{
				Cluster: config.ClusterConfig{
					EnvironmentGitPublic: true,
					GitPublic:            true,
					Provider:             cloud.GKE,
				},
			},
			ValidationFunc: func(orig *config.RequirementsConfig, ch *config.RequirementsConfig) {
				assert.True(t, orig.Cluster.EnvironmentGitPublic == ch.Cluster.EnvironmentGitPublic &&
					orig.Cluster.GitPublic == ch.Cluster.GitPublic &&
					orig.Cluster.ProjectID != ch.Cluster.ProjectID, "ClusterConfig validation failed")
			},
		},
		{
			Name: "Merge EnvironmentConfig slices Test",
			Original: &config.RequirementsConfig{
				Environments: []config.EnvironmentConfig{
					{
						Key:        "dev",
						Repository: "repo",
					},
					{
						Key: "production",
						Ingress: config.IngressConfig{
							Domain: "domain",
						},
					},
					{
						Key: "staging",
						Ingress: config.IngressConfig{
							Domain: "domainStaging",
							TLS: config.TLSConfig{
								Email: "email",
							},
						},
					},
				},
			},
			Changed: &config.RequirementsConfig{
				Environments: []config.EnvironmentConfig{
					{
						Key:   "dev",
						Owner: "owner",
					},
					{
						Key: "production",
						Ingress: config.IngressConfig{
							CloudDNSSecretName: "secret",
						},
					},
					{
						Key: "staging",
						Ingress: config.IngressConfig{
							Domain:          "newDomain",
							DomainIssuerURL: "issuer",
							TLS: config.TLSConfig{
								Enabled: true,
							},
						},
					},
					{
						Key: "ns2",
					},
				},
			},
			ValidationFunc: func(orig *config.RequirementsConfig, ch *config.RequirementsConfig) {
				assert.True(t, len(orig.Environments) == len(ch.Environments), "the environment slices should be of the same len")
				// -- Dev Environment's asserts
				devOrig, devCh := orig.Environments[0], ch.Environments[0]
				assert.True(t, devOrig.Owner == devCh.Owner && devOrig.Repository != devCh.Repository,
					"the dev environment should've been merged correctly")
				// -- Production Environment's asserts
				prOrig, prCh := orig.Environments[1], ch.Environments[1]
				assert.True(t, prOrig.Ingress.Domain == "domain" &&
					prOrig.Ingress.CloudDNSSecretName == prCh.Ingress.CloudDNSSecretName,
					"the production environment should've been merged correctly")
				// -- Staging Environmnet's asserts
				stgOrig, stgCh := orig.Environments[2], ch.Environments[2]
				assert.True(t, stgOrig.Ingress.Domain == stgCh.Ingress.Domain &&
					stgOrig.Ingress.TLS.Email == "email" && stgOrig.Ingress.TLS.Enabled == stgCh.Ingress.TLS.Enabled,
					"the staging environment should've been merged correctly")
			},
		},
		{
			Name: "Merge StorageConfig test",
			Original: &config.RequirementsConfig{
				Storage: config.StorageConfig{
					Logs: config.StorageEntryConfig{
						Enabled: true,
						URL:     "value1",
					},
					Reports: config.StorageEntryConfig{},
					Repository: config.StorageEntryConfig{
						Enabled: true,
						URL:     "value3",
					},
				},
			},
			Changed: &config.RequirementsConfig{
				Storage: config.StorageConfig{
					Reports: config.StorageEntryConfig{
						Enabled: true,
						URL:     "",
					},
				},
			},
			ValidationFunc: func(orig *config.RequirementsConfig, ch *config.RequirementsConfig) {
				assert.True(t, orig.Storage.Logs.Enabled && orig.Storage.Logs.URL == "value1" &&
					orig.Storage.Repository.Enabled && orig.Storage.Repository.URL == "value3" &&
					orig.Storage.Reports.Enabled == ch.Storage.Reports.Enabled,
					"The storage configuration should've been merged correctly")
			},
		},
	}
	f, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	defer util.DeleteFile(f.Name())

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err = tc.Original.MergeSave(tc.Changed, f.Name())
			assert.NoError(t, err, "the merge shouldn't fail for case %s", tc.Name)
			tc.ValidationFunc(tc.Original, tc.Changed)
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

func TestHelmfileRequirementValues(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "test-requirements-config-")
	assert.NoError(t, err, "should create a temporary config dir")

	file := filepath.Join(dir, config.RequirementsConfigFileName)
	requirements := config.NewRequirementsConfig()
	requirements.Cluster.ClusterName = "jx_rocks"

	err = requirements.SaveConfig(file)
	assert.NoError(t, err, "failed to save file %s", file)
	assert.FileExists(t, file)
	valuesFile := filepath.Join(dir, config.RequirementsValuesFileName)

	valuesFileExists, err := util.FileExists(valuesFile)
	assert.False(t, valuesFileExists, "file %s should not exist", valuesFile)

	requirements.Helmfile = true
	err = requirements.SaveConfig(file)
	assert.NoError(t, err, "failed to save file %s", file)
	assert.FileExists(t, file)
	assert.FileExists(t, valuesFile, "file %s should exist", valuesFile)
}

func Test_LoadRequirementsConfig(t *testing.T) {
	t.Parallel()

	var gitPublicTests = []struct {
		requirementsPath   string
		createRequirements bool
	}{
		{"a", false},
		{"a/b", false},
		{"a/b/c", false},
		{"e", true},
		{"e/f", true},
		{"e/f/g", true},
	}

	for _, testCase := range gitPublicTests {
		t.Run(testCase.requirementsPath, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "jx-test-load-requirements-config")
			require.NoError(t, err, "failed to create tmp directory")
			defer func() {
				_ = os.RemoveAll(dir)
			}()

			testPath := filepath.Join(dir, testCase.requirementsPath)
			err = os.MkdirAll(testPath, 0700)
			require.NoError(t, err, "unable to create test path %s", testPath)

			var expectedRequirementsFile string
			if testCase.createRequirements {
				expectedRequirementsFile = filepath.Join(testPath, config.RequirementsConfigFileName)
				dummyRequirementsData := []byte("webhook: prow\n")
				err := ioutil.WriteFile(expectedRequirementsFile, dummyRequirementsData, 0600)
				require.NoError(t, err, "unable to write requirements file %s", expectedRequirementsFile)
			}

			requirements, requirementsFile, err := config.LoadRequirementsConfig(testPath, config.DefaultFailOnValidationError)
			if testCase.createRequirements {
				require.NoError(t, err)
				assert.Equal(t, expectedRequirementsFile, requirementsFile)
				assert.Equal(t, fmt.Sprintf("%s", requirements.Webhook), "prow")
			} else {
				require.Error(t, err)
				assert.Equal(t, "", requirementsFile)
				assert.Nil(t, requirements)
			}
		})
	}
}

func TestLoadRequirementsConfig_load_invalid_yaml(t *testing.T) {
	testDir := path.Join(testDataDir, "jx-requirements-syntax-error")

	absolute, err := filepath.Abs(testDir)
	assert.NoError(t, err, "could not find absolute path of dir %s", testDataDir)

	_, _, err = config.LoadRequirementsConfig(testDir, config.DefaultFailOnValidationError)
	requirementsConfigPath := path.Join(absolute, config.RequirementsConfigFileName)
	assert.EqualError(t, err, fmt.Sprintf("validation failures in YAML file %s:\nenvironments.0: Additional property namespace is not allowed", requirementsConfigPath))
}
