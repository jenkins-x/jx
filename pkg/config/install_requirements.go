package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

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
	SecretStorageTypeVault SecretStorageType = "Vault"
	// SecretStorageTypeLocal specifies that we use the local file system in
	// `~/.jx/localSecrets` to store secretst
	SecretStorageTypeLocal SecretStorageType = "Local"
)

// RequirementsConfig contains the logical installation requirements
type RequirementsConfig struct {
	// Kaniko whether to enable kaniko for building docker images
	Kaniko bool `json:"kaniko,omitempty"`
	// Terraform specifies if  we are managing the kubernetes cluster and cloud resources with Terraform
	Terraform bool `json:"terraform,omitempty"`
	// SecretStorage how should we store secrets for the cluster
	SecretStorage SecretStorageType `json:"secretStorage,omitempty"`
	// ClusterName the logical name of the cluster
	ClusterName string `json:"clusterName,omitempty"`
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
