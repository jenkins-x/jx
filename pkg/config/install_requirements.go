package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
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

	semanticVersionRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*)?(\+[0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*)?$`)
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
	// RequirementTerraform if Terraform was used to create the cluster
	RequirementTerraform = "JX_REQUIREMENT_TERRAFORM"
	// RequirementClusterName is the cluster name
	RequirementClusterName = "JX_REQUIREMENT_CLUSTER_NAME"
	// RequirementProject is the cloudprovider project
	RequirementProject = "JX_REQUIREMENT_PROJECT"
	// RequirementZone zone the cluster is in
	RequirementZone = "JX_REQUIREMENT_ZONE"
	// RequirementEnvGitOwner the default git owner for environment repositories if none is specified explicitly
	RequirementEnvGitOwner = "JX_REQUIREMENT_ENV_GIT_OWNER"
	// RequirementExternalDNSServiceAccountName the service account name for external dns
	RequirementExternalDNSServiceAccountName = "JX_REQUIREMENT_EXTERNALDNS_SA_NAME"
	// RequirementVaultServiceAccountName the service account name for vault
	RequirementVaultServiceAccountName = "JX_REQUIREMENT_VAULT_SA_NAME"
	// RequirementKanikoServiceAccountName the service account name for kaniko
	RequirementKanikoServiceAccountName = "JX_REQUIREMENT_KANIKO_SA_NAME"
	// RequirementKaniko if kaniko is required
	RequirementKaniko = "JX_REQUIREMENT_KANIKO"
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
}

// IngressConfig contains dns specific requirements
type IngressConfig struct {
	// DNS is enabled
	ExternalDNS bool `json:"externalDNS"`
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
	// Repository for storing build logs
	Repository StorageEntryConfig `json:"repository"`
}

// ClusterConfig contains cluster specific requirements
type ClusterConfig struct {
	// EnvironmentGitOwner the default git owner for environment repositories if none is specified explicitly
	EnvironmentGitOwner string `json:"environmentGitOwner,omitempty"`
	// EnvironmentGitPrivate will request jx boot create private git repos for the environments
	EnvironmentGitPrivate bool `json:"environmentGitPrivate,omitempty"`
	// Provider the kubernetes provider (e.g. gke)
	Provider string `json:"provider,omitempty"`
	// Namespace the namespace to install the dev environment
	Namespace string `json:"namespace,omitempty"`
	// ProjectID the cloud project ID e.g. on GCP
	ProjectID string `json:"project,omitempty"`
	// ClusterName the logical name of the cluster
	ClusterName string `json:"clusterName,omitempty"`
	// VaultName the name of the vault if using vault for secretts
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
	// VaultSAName the service account name for vault
	VaultSAName string `json:"vaultSAName,omitempty"`
	// KanikoSAName the service account name for kaniko
	KanikoSAName string `json:"kanikoSAName,omitempty"`
}

// VersionStreamConfig contains version stream config
type VersionStreamConfig struct {
	// URL of the version stream to use
	URL string `json:"url"`
	// Ref of the version stream to use
	Ref string `json:"ref"`
}

// RequirementsConfig contains the logical installation requirements
type RequirementsConfig struct {
	// Cluster contains cluster specific requirements
	Cluster ClusterConfig `json:"cluster"`
	// Kaniko whether to enable kaniko for building docker images
	Kaniko bool `json:"kaniko,omitempty"`
	// GitOps if enabled we will setup a webhook in the boot configuration git repository so that we can
	// re-run 'jx boot' when changes merge to the master branch
	GitOps bool `json:"gitops,omitempty"`
	// Terraform specifies if  we are managing the kubernetes cluster and cloud resources with Terraform
	Terraform bool `json:"terraform,omitempty"`
	// SecretStorage how should we store secrets for the cluster
	SecretStorage SecretStorageType `json:"secretStorage,omitempty"`
	// Webhook specifies what engine we should use for webhooks
	Webhook WebhookType `json:"webhook,omitempty"`
	// Environments the requirements for the environments
	Environments []EnvironmentConfig `json:"environments,omitempty"`
	// Ingress contains ingress specific requirements
	Ingress IngressConfig `json:"ingress"`
	// Storage contains storage requirements
	Storage StorageConfig `json:"storage"`
	// VersionStream contains version stream info
	VersionStream VersionStreamConfig `json:"versionStream"`
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
	return config, nil
}

// IsEmpty returns true if this configuration is empty
func (c *RequirementsConfig) IsEmpty() bool {
	empty := &RequirementsConfig{}
	return reflect.DeepEqual(empty, c)
}

// SaveConfig saves the configuration file to the given project directory
func (c *RequirementsConfig) SaveConfig(fileName string) error {
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
