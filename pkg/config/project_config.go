package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
)

const (
	// ProjectConfigFileName is the name of the project configuration file
	ProjectConfigFileName = "jenkins-x.yml"
)

type ProjectConfig struct {
	PreviewEnvironments *PreviewEnvironmentConfig `yaml:"previewEnvironments,omitempty"`
	IssueTracker        *IssueTrackerConfig       `yaml:"issueTracker,omitempty"`
	Chat                *ChatConfig               `yaml:"chat,omitempty"`
	Wiki                *WikiConfig               `yaml:"wiki,omitempty"`
}

type PreviewEnvironmentConfig struct {
	Disabled         bool `yaml:"disabled,omitempty"`
	MaximumInstances int  `yaml:"maximumInstances,omitempty"`
}

type IssueTrackerConfig struct {
	Kind    string `yaml:"kind,omitempty"`
	URL     string `yaml:"url,omitempty"`
	Project string `yaml:"project,omitempty"`
}

type WikiConfig struct {
	Kind  string `yaml:"kind,omitempty"`
	URL   string `yaml:"url,omitempty"`
	Space string `yaml:"space,omitempty"`
}

type ChatConfig struct {
	Kind string `yaml:"kind,omitempty"`
	URL  string `yaml:"url,omitempty"`
	Room string `yaml:"room,omitempty"`
}

// LoadProjectConfig loads the project configuration if there is a project configuration file
func LoadProjectConfig(projectDir string) (*ProjectConfig, string, error) {
	fileName := ProjectConfigFileName
	if projectDir != "" {
		fileName = filepath.Join(projectDir, fileName)
	}
	config := ProjectConfig{}
	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		return &config, fileName, err
	}
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return &config, fileName, fmt.Errorf("Failed to load file %s due to %s", fileName, err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return &config, fileName, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
	}
	return &config, fileName, nil
}

// IsEmpty returns true if this configuration is empty
func (c *ProjectConfig) IsEmpty() bool {
	empty := ProjectConfig{}
	return *c == empty
}

// SaveConfig saves the configuration file to the given project directory
func (c *ProjectConfig) SaveConfig(fileName string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
}
