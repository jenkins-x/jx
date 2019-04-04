package config

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"io/ioutil"
	"path/filepath"
	"reflect"

	"github.com/jenkins-x/jx/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ProjectConfigFileName is the name of the project configuration file
	ProjectConfigFileName = "jenkins-x.yml"
)

type ProjectConfig struct {
	// List of global environment variables to add to each branch build and each step
	Env []corev1.EnvVar `json:"env,omitempty"`

	PreviewEnvironments *PreviewEnvironmentConfig   `json:"previewEnvironments,omitempty"`
	IssueTracker        *IssueTrackerConfig         `json:"issueTracker,omitempty"`
	Chat                *ChatConfig                 `json:"chat,omitempty"`
	Wiki                *WikiConfig                 `json:"wiki,omitempty"`
	Addons              []*AddonConfig              `json:"addons,omitempty"`
	BuildPack           string                      `json:"buildPack,omitempty"`
	BuildPackGitURL     string                      `json:"buildPackGitURL,omitempty"`
	BuildPackGitURef    string                      `json:"buildPackGitRef,omitempty"`
	Workflow            string                      `json:"workflow,omitempty"`
	PipelineConfig      *jenkinsfile.PipelineConfig `json:"pipelineConfig,omitempty"`
	NoReleasePrepare    bool                        `json:"noReleasePrepare,omitempty"`
}

type PreviewEnvironmentConfig struct {
	Disabled         bool `json:"disabled,omitempty"`
	MaximumInstances int  `json:"maximumInstances,omitempty"`
}

type IssueTrackerConfig struct {
	Kind    string `json:"kind,omitempty"`
	URL     string `json:"url,omitempty"`
	Project string `json:"project,omitempty"`
}

type WikiConfig struct {
	Kind  string `json:"kind,omitempty"`
	URL   string `json:"url,omitempty"`
	Space string `json:"space,omitempty"`
}

type ChatConfig struct {
	Kind             string `json:"kind,omitempty"`
	URL              string `json:"url,omitempty"`
	DeveloperChannel string `json:"developerChannel,omitempty"`
	UserChannel      string `json:"userChannel,omitempty"`
}

type AddonConfig struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type BranchBuild struct {
	Build Build `json:"build,omitempty"`

	// Jenkins X extensions to standard Knative builds:

	// which kind of pipeline - like release, pullRequest, feature
	Kind string `json:"kind,omitempty"`

	// display name
	Name string `json:"name,omitempty"`

	// List of sources to populate environment variables in all the steps if there is not already
	// an environment variable defined on that step
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// List of environment variables to add to each step if there is not already a environment variable of that name
	Env []corev1.EnvVar `json:"env,omitempty"`

	ExcludePodTemplateEnv     bool `json:"excludePodTemplateEnv,omitempty"`
	ExcludePodTemplateVolumes bool `json:"excludePodTemplateVolumes,omitempty"`
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
	config, err := LoadProjectConfigFile(fileName)
	return config, fileName, err
}

// LoadProjectConfigFile loads a specific project YAML configuration file
func LoadProjectConfigFile(fileName string) (*ProjectConfig, error) {
	config := ProjectConfig{}
	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		return &config, err
	}
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return &config, fmt.Errorf("Failed to load file %s due to %s", fileName, err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return &config, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
	}
	return &config, nil
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
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}
	return nil
}

// GetOrCreatePipelineConfig lazily creates a PipelineConfig if required
func (c *ProjectConfig) GetOrCreatePipelineConfig() *jenkinsfile.PipelineConfig {
	if c.PipelineConfig == nil {
		c.PipelineConfig = &jenkinsfile.PipelineConfig{}
	}
	return c.PipelineConfig
}
