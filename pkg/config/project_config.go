package config

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
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
	// List of global environment variables to add to each branch build and each step
	Env []corev1.EnvVar `yaml:"env,omitempty"`

	Builds              []*BranchBuild              `yaml:"builds,omitempty"`
	PreviewEnvironments *PreviewEnvironmentConfig   `yaml:"previewEnvironments,omitempty"`
	IssueTracker        *IssueTrackerConfig         `yaml:"issueTracker,omitempty"`
	Chat                *ChatConfig                 `yaml:"chat,omitempty"`
	Wiki                *WikiConfig                 `yaml:"wiki,omitempty"`
	Addons              []*AddonConfig              `yaml:"addons,omitempty"`
	BuildPack           string                      `yaml:"buildPack,omitempty"`
	BuildPackGitURL     string                      `yaml:"buildPackGitURL,omitempty"`
	BuildPackGitURef    string                      `yaml:"buildPackGitRef,omitempty"`
	Workflow            string                      `yaml:"workflow,omitempty"`
	PipelineConfig      *jenkinsfile.PipelineConfig `yaml:"pipelineConfig,omitempty"`
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
	Build Build `yaml:"build,omitempty"`

	// Jenkins X extensions to standard Knative builds:

	// which kind of pipeline - like release, pullRequest, feature
	Kind string `yaml:"kind,omitempty"`

	// display name
	Name string `yaml:"name,omitempty"`

	// List of sources to populate environment variables in all the steps if there is not already
	// an environment variable defined on that step
	EnvFrom []corev1.EnvFromSource `yaml:"envFrom,omitempty"`

	// List of environment variables to add to each step if there is not already a environemnt variable of that name
	Env []corev1.EnvVar `yaml:"env,omitempty"`

	ExcludePodTemplateEnv     bool `yaml:"excludePodTemplateEnv,omitempty"`
	ExcludePodTemplateVolumes bool `yaml:"excludePodTemplateVolumes,omitempty"`
}

type Build struct {
	// Steps are the steps of the build; each step is run sequentially with the
	// source mounted into /workspace.
	Steps []corev1.Container `yaml:"steps,omitempty"`

	// Volumes is a collection of volumes that are available to mount into the
	// steps of the build.
	Volumes []corev1.Volume `yaml:"volumes,omitempty"`

	// The name of the service account as which to run this build.
	ServiceAccountName string `yaml:"serviceAccountName,omitempty"`

	// Template, if specified, references a BuildTemplate resource to use to
	// populate fields in the build, and optional Arguments to pass to the
	// template.
	//Template *TemplateInstantiationSpec `yaml:"template,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty"`
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
