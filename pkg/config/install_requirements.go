package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"

	"io/ioutil"
	"path/filepath"
	"reflect"

	"github.com/jenkins-x/jx/pkg/util"
)

const (
	// RequirementsConfigFileName is the name of the requirements configuration file
	RequirementsConfigFileName = "jx-requirements.yml"
)

// SecretStorageType is the type of a promotion strategy
type SecretStorageType string

const (
	// SecretStorageTypeVault specifies that we use vault to store secrets
	SecretStorageTypeVault SecretStorageType = "vault"
	// SecretStorageTypeLocal specifies that we use the local file system in
	// `~/.jx/localSecrets` to store secrets
	SecretStorageTypeLocal SecretStorageType = "local"
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

// RequirementsConfig contains the logical installation requirements
type RequirementsConfig struct {
	// Kaniko whether to enable kaniko for building docker images
	Kaniko bool `json:"kaniko,omitempty"`
	// Terraform specifies if  we are managing the kubernetes cluster and cloud resources with Terraform
	Terraform bool `json:"terraform,omitempty"`
	// SecretStorage how should we store secrets for the cluster
	SecretStorage SecretStorageType `json:"secretStorage,omitempty"`
	// Provider the kubernetes provider (e.g. gke)
	Provider string `json:"provider,omitempty"`
	// ProjectID the cloud project ID e.g. on GCP
	ProjectID string `json:"project,omitempty"`
	// ClusterName the logical name of the cluster
	ClusterName string `json:"clusterName,omitempty"`
	// Region the cloud region being used
	Region string `json:"region,omitempty"`
	// Zone the cloud zone being used
	Zone string `json:"zone,omitempty"`
	// EnvironmentGitOwner the default git owner for environment repositories if none is specified explicitly
	EnvironmentGitOwner string `json:"environmentGitOwner,omitempty"`
	// Environments the requirements for the environments
	Environments []EnvironmentConfig `json:"environments,omitempty"`
}

// NewRequirementsConfig creates a default configuration file
func NewRequirementsConfig() *RequirementsConfig {
	return &RequirementsConfig{
		SecretStorage: SecretStorageTypeLocal,
		Kaniko:        true,
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
	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		path, err := filepath.Abs(fileName)
		if err != nil {
			config, _ := LoadRequirementsConfigFile(fileName)
			return config, fileName, err
		}
		subDir := GetParentDir(path)

		// lets walk up the directory tree to see if we can find a requirements file in a parent dir
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
	}
	config, err := LoadRequirementsConfigFile(fileName)
	return config, fileName, err
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
		m, err := toObjectMap(&env)
		if err == nil {
			ensureHasFields(m, "owner", "repository", "gitServer", "gitKind")
			answer[k] = m
		} else {
			log.Logger().Warnf("failed to turn environment %s with value %#v into a map: %s\n", k, env, err.Error())
		}
	}
	log.Logger().Infof("Enviroments: %#v\n", answer)
	return answer
}

func ensureHasFields(m map[string]interface{}, keys ...string) {
	for _, k := range keys {
		_, ok := m[k]
		if !ok {
			m[k] = ""
		}
	}
}

// toObjectMap converts the given object into a map of strings/maps using YAML marshalling
func toObjectMap(object interface{}) (map[string]interface{}, error) {
	answer := map[string]interface{}{}
	data, err := yaml.Marshal(object)
	if err != nil {
		return answer, err
	}
	err = yaml.Unmarshal(data, &answer)
	return answer, err
}

// MissingRequirement returns an error if there is a missing property in the requirements
func MissingRequirement(property string, fileName string) error {
	return fmt.Errorf("missing property: %s in file %s", property, fileName)
}
