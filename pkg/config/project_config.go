package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ProjectConfigFileName is the name of the project configuration file
	ProjectConfigFileName = "jenkins-x.yml"
)

type ProjectConfig struct {
	Builds              []*BranchBuild            `yaml:"builds,omitempty"`
	PreviewEnvironments *PreviewEnvironmentConfig `yaml:"previewEnvironments,omitempty"`
	IssueTracker        *IssueTrackerConfig       `yaml:"issueTracker,omitempty"`
	Chat                *ChatConfig               `yaml:"chat,omitempty"`
	Wiki                *WikiConfig               `yaml:"wiki,omitempty"`
	Addons              []*AddonConfig            `yaml:"addons,omitempty"`
	BuildPack           string                    `yaml:"buildPack,omitempty"`
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
	Kind             string `yaml:"kind,omitempty"`
	URL              string `yaml:"url,omitempty"`
	DeveloperChannel string `yaml:"developerChannel,omitempty"`
	UserChannel      string `yaml:"userChannel,omitempty"`
}

type AddonConfig struct {
	Name    string `yaml:"name,omitempty"`
	Version string `yaml:"version,omitempty"`
}

type BranchBuild struct {
	BranchKind string `yaml:"branchKind,omitempty"`
	Build      Build  `yaml:"version,omitempty"`
}

type Build struct {
	// Steps are the steps of the build; each step is run sequentially with the
	// source mounted into /workspace.
	Steps []corev1.Container `json:"steps,omitempty"`

	// Volumes is a collection of volumes that are available to mount into the
	// steps of the build.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// The name of the service account as which to run this build.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Template, if specified, references a BuildTemplate resource to use to
	// populate fields in the build, and optional Arguments to pass to the
	// template.
	//Template *TemplateInstantiationSpec `json:"template,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
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
	empty := &ProjectConfig{}
	return reflect.DeepEqual(empty, c)
}

// SaveConfig saves the configuration file to the given project directory
func (c *ProjectConfig) SaveConfig(fileName string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
}
