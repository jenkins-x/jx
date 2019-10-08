package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"

	"io/ioutil"
	"path/filepath"
	"reflect"

	"github.com/jenkins-x/jx/pkg/util"
)

var (
	// autoDNSSuffixes the DNS suffixes of any auto-DNS services
	autoDNSSuffixes = []string{
		".nip.io",
		".xip.io",
		".beesdns.com",
	}
)

const (
	// RequirementsConfigFileName is the name of the requirements configuration file
	RequirementsConfigFileName = "jx-requirements.yml"
	// RequirementDomainIssuerUsername contains the username used for basic auth when requesting a domain
	RequirementDomainIssuerUsername = "JX_REQUIREMENT_DOMAIN_ISSUER_USERNAME"
	// RequirementDomainIssuerPassword contains the password used for basic auth when requesting a domain
	RequirementDomainIssuerPassword = "JX_REQUIREMENT_DOMAIN_ISSUER_PASSWORD"
	// RequirementDomainIssuerURL contains the URL to the service used when requesting a domain
	RequirementDomainIssuerURL = "JX_REQUIREMENT_DOMAIN_ISSUER_URL"
	// RequirementClusterName is the cluster name
	RequirementClusterName = "JX_REQUIREMENT_CLUSTER_NAME"
	// RequirementProject is the cloudprovider project
	RequirementProject = "JX_REQUIREMENT_PROJECT"
	// RequirementZone zone the cluster is in
	RequirementZone = "JX_REQUIREMENT_ZONE"
	// RequirementEnvGitOwner the default git owner for environment repositories if none is specified explicitly
	RequirementEnvGitOwner = "JX_REQUIREMENT_ENV_GIT_OWNER"
	// RequirementEnvGitPublic sets the visibility of the environment repositories as private (subscription required for GitHub Organisations)
	RequirementEnvGitPublic = "JX_REQUIREMENT_ENV_GIT_PUBLIC"
	// RequirementExternalDNSServiceAccountName the service account name for external dns
	RequirementExternalDNSServiceAccountName = "JX_REQUIREMENT_EXTERNALDNS_SA_NAME"
	// RequirementVaultName the name for vault
	RequirementVaultName = "JX_REQUIREMENT_VAULT_NAME"
	// RequirementVaultServiceAccountName the service account name for vault
	RequirementVaultServiceAccountName = "JX_REQUIREMENT_VAULT_SA_NAME"
	// RequirementVaultKeyringName the keyring name for vault
	RequirementVaultKeyringName = "JX_REQUIREMENT_VAULT_KEYRING_NAME"
	// RequirementVaultKeyName the key name for vault
	RequirementVaultKeyName = "JX_REQUIREMENT_VAULT_KEY_NAME"
	// RequirementVaultBucketName the vault name for vault
	RequirementVaultBucketName = "JX_REQUIREMENT_VAULT_BUCKET_NAME"
	// RequirementVaultRecreateBucket recreate the bucket that vault uses
	RequirementVaultRecreateBucket = "JX_REQUIREMENT_VAULT_RECREATE_BUCKET"
	// RequirementVaultDisableURLDiscovery override the default lookup of the Vault URL, could be incluster service or external ingress
	RequirementVaultDisableURLDiscovery = "JX_REQUIREMENT_VAULT_DISABLE_URL_DISCOVERY"
	// RequirementSecretStorageType the secret storage type
	RequirementSecretStorageType = "JX_REQUIREMENT_SECRET_STORAGE_TYPE"
	// RequirementKanikoServiceAccountName the service account name for kaniko
	RequirementKanikoServiceAccountName = "JX_REQUIREMENT_KANIKO_SA_NAME"
	// RequirementKaniko if kaniko is required
	RequirementKaniko = "JX_REQUIREMENT_KANIKO"
	// RequirementIngressTLSProduction use the lets encrypt production server
	RequirementIngressTLSProduction = "JX_REQUIREMENT_INGRESS_TLS_PRODUCTION"
	// RequirementWebhook the webhook handler for jx
	RequirementWebhook = "JX_REQUIREMENT_WEBHOOK"
	// RequirementStorageBackupEnabled if backup storage is required
	RequirementStorageBackupEnabled = "JX_REQUIREMENT_STORAGE_BACKUP_ENABLED"
	// RequirementStorageBackupURL backup storage url
	RequirementStorageBackupURL = "JX_REQUIREMENT_STORAGE_BACKUP_URL"
	// RequirementStorageLogsEnabled if log storage is required
	RequirementStorageLogsEnabled = "JX_REQUIREMENT_STORAGE_LOGS_ENABLED"
	// RequirementStorageLogsURL logs storage url
	RequirementStorageLogsURL = "JX_REQUIREMENT_STORAGE_LOGS_URL"
	// RequirementStorageReportsEnabled if report storage is required
	RequirementStorageReportsEnabled = "JX_REQUIREMENT_STORAGE_REPORTS_ENABLED"
	// RequirementStorageReportsURL report storage url
	RequirementStorageReportsURL = "JX_REQUIREMENT_STORAGE_REPORTS_URL"
	// RequirementStorageRepositoryEnabled if repository storage is required
	RequirementStorageRepositoryEnabled = "JX_REQUIREMENT_STORAGE_REPOSITORY_ENABLED"
	// RequirementStorageRepositoryURL repository storage url
	RequirementStorageRepositoryURL = "JX_REQUIREMENT_STORAGE_REPOSITORY_URL"
)

// SecretStorageType is the type of storage used for secrets
type SecretStorageType string

const (
	// SecretStorageTypeVault specifies that we use vault to store secrets
	SecretStorageTypeVault SecretStorageType = "vault"
	// SecretStorageTypeLocal specifies that we use the local file system in
	// `~/.jx/localSecrets` to store secrets
	SecretStorageTypeLocal SecretStorageType = "local"
)

// SecretStorageTypeValues the string values for the secret storage
var SecretStorageTypeValues = []string{"local", "vault"}

// WebhookType is the type of a webhook strategy
type WebhookType string

const (
	// WebhookTypeNone if we have yet to define a webhook
	WebhookTypeNone WebhookType = ""
	// WebhookTypeProw specifies that we use prow for webhooks
	// see: https://github.com/kubernetes/test-infra/tree/master/prow
	WebhookTypeProw WebhookType = "prow"
	// WebhookTypeLighthouse specifies that we use lighthouse for webhooks
	// see: https://github.com/jenkins-x/lighthouse
	WebhookTypeLighthouse WebhookType = "lighthouse"
	// WebhookTypeJenkins specifies that we use jenkins webhooks
	WebhookTypeJenkins WebhookType = "jenkins"
)

// WebhookTypeValues the string values for the webhook types
var WebhookTypeValues = []string{"jenkins", "lighthouse", "prow"}

const (
	// DefaultVersionsURL default version stream url
	DefaultVersionsURL = "https://github.com/jenkins-x/jenkins-x-versions.git"
	// DefaultVersionsRef default version stream ref
	DefaultVersionsRef = "master"
	// DefaultProfileFile location of profle config
	DefaultProfileFile = "profile.yaml"
	// OpenSourceProfile constant for OSS profile
	OpenSourceProfile = "oss"
	// CloudBeesProfile constant for CloudBees profile
	CloudBeesProfile = "cloudbees"
	// DefaultBootRepository default git repo for boot
	DefaultBootRepository = "https://github.com/jenkins-x/jenkins-x-boot-config.git"
	// DefaultCloudBeesBootRepository boot git repo to use when using cloudbees profile
	DefaultCloudBeesBootRepository = "https://github.com/cloudbees/cloudbees-jenkins-x-boot-config.git"
	// DefaultCloudBeesVersionsURL default version stream url for cloudbees jenkins x distribution
	DefaultCloudBeesVersionsURL = "https://github.com/cloudbees/cloudbees-jenkins-x-versions.git"
	// DefaultCloudBeesVersionsRef default version stream ref for cloudbees jenkins x distribution
	DefaultCloudBeesVersionsRef = "master"
)

// EnvironmentConfig configures the organisation and repository name of the git repositories for environments
type EnvironmentConfig struct {
	// Key is the key of the environment configuration
	Key string `json:"key,omitempty"`
	// Owner is the git user or organisation for the repository
	Owner string `json:"owner,omitempty"`
	// Repository is the name of the repository within the owner
	Repository string `json:"repository,omitempty"`
	// GitServer is the URL of the git server
	GitServer string `json:"gitServer,omitempty"`
	// GitKind is the kind of git server (github, bitbucketserver etc)
	GitKind string `json:"gitKind,omitempty"`
	// Ingress contains ingress specific requirements
	Ingress IngressConfig `json:"ingress,omitempty"`
	// RemoteCluster specifies this environment runs on a remote cluster to the development cluster
	RemoteCluster bool `json:"remotetCluster,omitempty"`
}

// IngressConfig contains dns specific requirements
type IngressConfig struct {
	// DNS is enabled
	ExternalDNS bool `json:"externalDNS"`
	// CloudDNSSecretName secret name which contains the service account for external-dns and cert-manager issuer to
	// access the Cloud DNS service to resolve a DNS challenge
	CloudDNSSecretName string `json:"cloud_dns_secret_name,omitempty"`
	// Domain to expose ingress endpoints
	Domain string `json:"domain"`
	// NamespaceSubDomain the sub domain expression to expose ingress. Defaults to ".jx."
	NamespaceSubDomain string `json:"namespaceSubDomain"`
	// TLS enable automated TLS using certmanager
	TLS TLSConfig `json:"tls"`
	// DomainIssuerURL contains a URL used to retrieve a Domain
	DomainIssuerURL string `json:"domainIssuerURL,omitempty"`
}

// TLSConfig contains TLS specific requirements
type TLSConfig struct {
	// TLS enabled
	Enabled bool `json:"enabled"`
	// Email address to register with services like LetsEncrypt
	Email string `json:"email"`
	// Production false uses self-signed certificates from the LetsEncrypt staging server, true enables the production
	// server which incurs higher rate limiting https://letsencrypt.org/docs/rate-limits/
	Production bool `json:"production"`
}

// JxInstallProfile contains the jx profile info
type JxInstallProfile struct {
	InstallType string
}

// StorageEntryConfig contains dns specific requirements for a kind of storage
type StorageEntryConfig struct {
	// Enabled if the storage is enabled
	Enabled bool `json:"enabled"`
	// URL the cloud storage bucket URL such as 'gs://mybucket' or 's3://foo' or `azblob://thingy'
	// see https://jenkins-x.io/architecture/storage/
	URL string `json:"url"`
}

// StorageConfig contains dns specific requirements
type StorageConfig struct {
	// Logs for storing build logs
	Logs StorageEntryConfig `json:"logs"`
	// Tests for storing test results, coverage + code quality reports
	Reports StorageEntryConfig `json:"reports"`
	// Repository for storing repository artifacts
	Repository StorageEntryConfig `json:"repository"`
	// Backup for backing up kubernetes resource
	Backup StorageEntryConfig `json:"backup"`
}

// AzureConfig contains Azure specific requirements
type AzureConfig struct {
	// RegistrySubscription the registry subscription for defaulting the container registry.
	// Not used if you specify a Registry explicitly
	RegistrySubscription string `json:"registrySubscription,omitempty"`
}

// ClusterConfig contains cluster specific requirements
type ClusterConfig struct {
	// AzureConfig the azure specific configuration
	AzureConfig AzureConfig `json:"azure,omitempty"`
	// EnvironmentGitOwner the default git owner for environment repositories if none is specified explicitly
	EnvironmentGitOwner string `json:"environmentGitOwner,omitempty"`
	// EnvironmentGitPublic determines whether jx boot create public or private git repos for the environments
	EnvironmentGitPublic bool `json:"environmentGitPublic,omitempty"`
	// Provider the kubernetes provider (e.g. gke)
	Provider string `json:"provider,omitempty"`
	// Namespace the namespace to install the dev environment
	Namespace string `json:"namespace,omitempty"`
	// ProjectID the cloud project ID e.g. on GCP
	ProjectID string `json:"project,omitempty"`
	// ClusterName the logical name of the cluster
	ClusterName string `json:"clusterName,omitempty"`
	// VaultName the name of the vault if using vault for secrets
	// Deprecated
	VaultName string `json:"vaultName,omitempty"`
	// Region the cloud region being used
	Region string `json:"region,omitempty"`
	// Zone the cloud zone being used
	Zone string `json:"zone,omitempty"`
	// GitName is the name of the default git service
	GitName string `json:"gitName,omitempty"`
	// GitKind is the kind of git server (github, bitbucketserver etc)
	GitKind string `json:"gitKind,omitempty"`
	// GitServer is the URL of the git server
	GitServer string `json:"gitServer,omitempty"`
	// ExternalDNSSAName the service account name for external dns
	ExternalDNSSAName string `json:"externalDNSSAName,omitempty"`
	// Registry the host name of the container registry
	Registry string `json:"registry,omitempty"`
	// VaultSAName the service account name for vault
	// Deprecated
	VaultSAName string `json:"vaultSAName,omitempty"`
	// KanikoSAName the service account name for kaniko
	KanikoSAName string `json:"kanikoSAName,omitempty"`
	// HelmMajorVersion contains the major helm version number. Assumes helm 2.x with no tiller if no value specified
	HelmMajorVersion string `json:"helmMajorVersion,omitempty"`
}

type VaultConfig struct {
	// Name the name of the vault if using vault for secretts
	Name           string `json:"name,omitempty"`
	Bucket         string `json:"bucket,omitempty"`
	Keyring        string `json:"keyring,omitempty"`
	Key            string `json:"key,omitempty"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
	RecreateBucket bool   `json:"recreateBucket,omitempty"`
	// Optionally allow us to override the default lookup of the Vault URL, could be incluster service or external ingress
	DisableURLDiscovery bool `json:"disableURLDiscovery,omitempty"`
}

// UnmarshalJSON method handles the rename of EnvironmentGitPrivate to EnvironmentGitPublic.
func (t *ClusterConfig) UnmarshalJSON(data []byte) error {
	// need a type alias to go into infinite loop
	type Alias ClusterConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	_, gitPublicSet := raw["environmentGitPublic"]
	private, gitPrivateSet := raw["environmentGitPrivate"]

	if gitPrivateSet && gitPublicSet {
		return errors.New("found settings for EnvironmentGitPublic as well as EnvironmentGitPrivate in ClusterConfig, only EnvironmentGitPublic should be used")
	}

	if gitPrivateSet {
		log.Logger().Warn("EnvironmentGitPrivate specified in ClusterConfig. EnvironmentGitPrivate is deprecated use EnvironmentGitPublic instead.")
		privateString := string(private)
		if privateString == "true" {
			t.EnvironmentGitPublic = false
		} else {
			t.EnvironmentGitPublic = true
		}
	}
	return nil
}

// VersionStreamConfig contains version stream config
type VersionStreamConfig struct {
	// URL of the version stream to use
	URL string `json:"url"`
	// Ref of the version stream to use
	Ref string `json:"ref"`
}

// VeleroConfig contains the configuration for velero
type VeleroConfig struct {
	// Namespace the namespace to install velero into
	Namespace string `json:"namespace,omitempty"`
	// ServiceAccount the cloud service account used to run velero
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// AutoUpdateConfig contains auto update config
type AutoUpdateConfig struct {
	// Enabled autoupdate
	Enabled bool `json:"enabled"`
	// Schedule cron of auto updates
	Schedule string `json:"schedule"`
}

// RequirementsConfig contains the logical installation requirements
type RequirementsConfig struct {
	// AutoUpdate contains auto update config
	AutoUpdate AutoUpdateConfig `json:"autoUpdate,omitempty"`
	// BootConfigURL contains the url to which the dev environment is associated with
	BootConfigURL string `json:"bootConfigURL,omitempty"`
	// Cluster contains cluster specific requirements
	Cluster ClusterConfig `json:"cluster"`
	// Environments the requirements for the environments
	Environments []EnvironmentConfig `json:"environments,omitempty"`
	// GitOps if enabled we will setup a webhook in the boot configuration git repository so that we can
	// re-run 'jx boot' when changes merge to the master branch
	GitOps bool `json:"gitops,omitempty"`
	// Kaniko whether to enable kaniko for building docker images
	Kaniko bool `json:"kaniko,omitempty"`
	// Ingress contains ingress specific requirements
	Ingress IngressConfig `json:"ingress"`
	// SecretStorage how should we store secrets for the cluster
	SecretStorage SecretStorageType `json:"secretStorage,omitempty"`
	// Storage contains storage requirements
	Storage StorageConfig `json:"storage"`
	// Terraform specifies if  we are managing the kubernetes cluster and cloud resources with Terraform
	Terraform bool `json:"terraform,omitempty"`
	// Vault the configuration for vault
	Vault VaultConfig `json:"vault,omitempty"`
	// Velero the configuration for running velero for backing up the cluster resources
	Velero VeleroConfig `json:"velero,omitempty"`
	// VersionStream contains version stream info
	VersionStream VersionStreamConfig `json:"versionStream"`
	// Webhook specifies what engine we should use for webhooks
	Webhook WebhookType `json:"webhook,omitempty"`
}

// NewRequirementsConfig creates a default configuration file
func NewRequirementsConfig() *RequirementsConfig {
	return &RequirementsConfig{
		SecretStorage: SecretStorageTypeLocal,
		Webhook:       WebhookTypeProw,
	}
}

// LoadRequirementsConfig loads the project configuration if there is a project configuration file
// if there is not a file called `jx-requirements.yml` in the given dir we will scan up the parent
// directories looking for the requirements file as we often run 'jx' steps in sub directories.
func LoadRequirementsConfig(dir string) (*RequirementsConfig, string, error) {
	fileName := RequirementsConfigFileName
	if dir != "" {
		fileName = filepath.Join(dir, fileName)
	}
	originalFileName := fileName
	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		path, err := filepath.Abs(fileName)
		if err != nil {
			config, _ := LoadRequirementsConfigFile(fileName)
			return config, fileName, err
		}
		subDir := GetParentDir(path)

		// lets walk up the directory tree to see if we can find a requirements file in a parent dir
		// if by the end we have not found a requirements file lets use the original filename
		for {
			subDir = GetParentDir(subDir)
			if subDir == "" || subDir == "/" {
				break
			}
			fileName = filepath.Join(subDir, RequirementsConfigFileName)
			exists, _ := util.FileExists(fileName)
			if exists {
				config, err := LoadRequirementsConfigFile(fileName)
				return config, fileName, err
			}
		}
		// set back to the original filename
		fileName = originalFileName
	}
	config, err := LoadRequirementsConfigFile(fileName)
	return config, fileName, err
}

// LoadActiveInstallProfile loads the active install profile
func LoadActiveInstallProfile() string {
	jxHome, err := util.ConfigDir()
	if err == nil {
		profileSettingsFile := filepath.Join(jxHome, DefaultProfileFile)
		exists, err := util.FileExists(profileSettingsFile)
		if err == nil && exists {
			jxProfle := JxInstallProfile{}
			data, err := ioutil.ReadFile(profileSettingsFile)
			err = yaml.Unmarshal(data, &jxProfle)
			if err == nil {
				return jxProfle.InstallType
			}
		}
	}
	return OpenSourceProfile
}

// GetParentDir returns the parent directory without a trailing separator
func GetParentDir(path string) string {
	subDir, _ := filepath.Split(path)
	i := len(subDir) - 1
	if os.IsPathSeparator(subDir[i]) {
		subDir = subDir[0:i]
	}
	return subDir
}

// LoadRequirementsConfigFile loads a specific project YAML configuration file
func LoadRequirementsConfigFile(fileName string) (*RequirementsConfig, error) {
	config := NewRequirementsConfig()
	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		return config, err
	}
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
	config.addDefaults()
	config.handleDeprecation()
	return config, nil
}

// GetRequirementsConfigFromTeamSettings reads the BootRequirements string from TeamSettings and unmarshals it
func GetRequirementsConfigFromTeamSettings(settings *v1.TeamSettings) (*RequirementsConfig, error) {
	if settings == nil {
		return nil, nil
	}
	// TeamSettings does not have a real value for BootRequirements, so this is probably not a boot cluster.
	if settings.BootRequirements == "" {
		return nil, nil
	}

	config := &RequirementsConfig{}
	data := []byte(settings.BootRequirements)
	validationErrors, err := util.ValidateYaml(config, data)
	if err != nil {
		return config, fmt.Errorf("failed to validate requirements from team settings due to %s", err)
	}
	if len(validationErrors) > 0 {
		return config, fmt.Errorf("validation failures in requirements from team settings:\n%s", strings.Join(validationErrors, "\n"))
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal requirements from team settings due to %s", err)
	}
	return config, nil
}

// IsEmpty returns true if this configuration is empty
func (c *RequirementsConfig) IsEmpty() bool {
	empty := &RequirementsConfig{}
	return reflect.DeepEqual(empty, c)
}

// SaveConfig saves the configuration file to the given project directory
func (c *RequirementsConfig) SaveConfig(fileName string) error {
	c.handleDeprecation()
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}
	return nil
}

// EnvironmentMap creates a map of maps tree which can be used inside Go templates to access the environment
// configurations
func (c *RequirementsConfig) EnvironmentMap() map[string]interface{} {
	answer := map[string]interface{}{}
	for _, env := range c.Environments {
		k := env.Key
		if k == "" {
			log.Logger().Warnf("missing 'key' for Environment requirements %#v", env)
			continue
		}
		m, err := util.ToObjectMap(&env)
		if err == nil {
			ensureHasFields(m, "owner", "repository", "gitServer", "gitKind")
			answer[k] = m
		} else {
			log.Logger().Warnf("failed to turn environment %s with value %#v into a map: %s\n", k, env, err.Error())
		}
	}
	return answer
}

// Environment looks up the environment configuration based on environment name
func (c *RequirementsConfig) Environment(name string) (*EnvironmentConfig, error) {
	for _, env := range c.Environments {
		if env.Key == name {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("environment %q not found", name)
}

// ToMap converts this object to a map of maps for use in helm templating
func (c *RequirementsConfig) ToMap() (map[string]interface{}, error) {
	m, err := util.ToObjectMap(c)
	if m != nil {
		ensureHasFields(m, "provider", "project", "environmentGitOwner", "gitops", "webhook")
	}
	return m, err
}

// IsCloudProvider returns true if the kubenretes provider is a cloud
func (c *RequirementsConfig) IsCloudProvider() bool {
	p := c.Cluster.Provider
	return p == cloud.GKE || p == cloud.AKS || p == cloud.AWS || p == cloud.EKS || p == cloud.ALIBABA
}

func ensureHasFields(m map[string]interface{}, keys ...string) {
	for _, k := range keys {
		_, ok := m[k]
		if !ok {
			m[k] = ""
		}
	}
}

// MissingRequirement returns an error if there is a missing property in the requirements
func MissingRequirement(property string, fileName string) error {
	return fmt.Errorf("missing property: %s in file %s", property, fileName)
}

// IsLazyCreateSecrets returns a boolean whether secrets should be lazily created
func (c *RequirementsConfig) IsLazyCreateSecrets(flag string) (bool, error) {
	if flag != "" {
		if flag == "true" {
			return true, nil
		} else if flag == "false" {
			return false, nil
		} else {
			return false, util.InvalidOption("lazy-create", flag, []string{"true", "false"})
		}
	} else {
		// lets default from the requirements
		if !c.Terraform {
			return true, nil
		}
	}
	// default to false
	return false, nil
}

// addDefaults lets ensure any missing values have good defaults
func (c *RequirementsConfig) addDefaults() {
	if c.Cluster.Namespace == "" {
		c.Cluster.Namespace = "jx"
	}
	if c.Cluster.GitServer == "" {
		c.Cluster.GitServer = "https://github.com"
	}
	if c.Cluster.GitKind == "" {
		c.Cluster.GitKind = "github"
	}
	if c.Cluster.GitName == "" {
		c.Cluster.GitName = c.Cluster.GitKind
	}
	if c.Ingress.NamespaceSubDomain == "" {
		c.Ingress.NamespaceSubDomain = "-" + c.Cluster.Namespace + "."
	}
	if c.Webhook == WebhookTypeNone {
		if c.Cluster.GitServer == "https://github.com" || c.Cluster.GitServer == "https://github.com/" {
			c.Webhook = WebhookTypeProw
		} else {
			// TODO when lighthouse is GA lets default to it
			// c.Webhook = WebhookTypeLighthouse
		}
	}
}

func (c *RequirementsConfig) handleDeprecation() {
	if c.Vault.Name != "" {
		c.Cluster.VaultName = c.Vault.Name
	} else {
		c.Vault.Name = c.Cluster.VaultName
	}

	if c.Vault.ServiceAccount != "" {
		c.Cluster.VaultSAName = c.Vault.ServiceAccount
	} else {
		c.Vault.ServiceAccount = c.Cluster.VaultSAName
	}
}

// IsAutoDNSDomain returns true if the domain is configured to use an auto DNS sub domain like
// '.nip.io' or '.xip.io'
func (i *IngressConfig) IsAutoDNSDomain() bool {
	for _, suffix := range autoDNSSuffixes {
		if strings.HasSuffix(i.Domain, suffix) {
			return true
		}
	}
	return false
}
