package jenkinsfile

import (
	"bytes"
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"strings"
)

const (
	// PipelineConfigFileName is the name of the pipeline configuration file
	PipelineConfigFileName = "pipeline.yaml"

	// PipelineTemplateFileName defines the jenkisnfile template used to generate the pipeline
	PipelineTemplateFileName = "Jenkinsfile.tmpl"

	indent = "          "
)

// PipelineAgent contains the agent definition metadata
type PipelineAgent struct {
	Label string `yaml:"label,omitempty"`
}

// Pipelines contains all the different kinds of pipeline for diferent branches
type Pipelines struct {
	PullRequest *PipelineLifecycles `yaml:"pullRequest,omitempty"`
	Release     *PipelineLifecycles `yaml:"release,omitempty"`
	Feature     *PipelineLifecycles `yaml:"feature,omitempty"`
}

// PipelineLifecycle defines an individual step in a pipeline, either a command (sh) or groovy block
type PipelineStep struct {
	Comment   string          `yaml:"comment,omitempty"`
	Container string          `yaml:"container,omitempty"`
	Command   string          `yaml:"cmd,omitempty"`
	Groovy    string          `yaml:"groovy,omitempty"`
	Steps     []*PipelineStep `yaml:"steps,omitempty"`
}

// PipelineLifecycles defines the steps of a lifecycle section
type PipelineLifecycles struct {
	Setup      *PipelineLifecycle `yaml:"setup,omitempty"`
	SetVersion *PipelineLifecycle `yaml:"setVersion,omitempty"`
	PreBuild   *PipelineLifecycle `yaml:"preBuild,omitempty"`
	Build      *PipelineLifecycle `yaml:"build,omitempty"`
	PostBuild  *PipelineLifecycle `yaml:"postBuild,omitempty"`
	Promote    *PipelineLifecycle `yaml:"promote,omitempty"`
}

// PipelineLifecycle defines the steps of a lifecycle section
type PipelineLifecycle struct {
	Steps []*PipelineStep `yaml:"steps,omitempty"`
}

// PipelineConfig defines the pipeline configuration
type PipelineConfig struct {
	Agent       PipelineAgent `yaml:"agent,omitempty"`
	Environment string        `yaml:"environment,omitempty"`
	Pipelines   Pipelines     `yaml:"pipelines,omitempty"`
}

// CreateJenkinsfileArguments contains the arguents to generate a Jenkinsfiles dynamically
type CreateJenkinsfileArguments struct {
	ConfigFile   string
	TemplateFile string
	OutputFile   string
}

// Validate validates all the arguments are set correctly
func (a *CreateJenkinsfileArguments) Validate() error {
	if a.ConfigFile == "" {
		return fmt.Errorf("Missing argument: ConfigFile")
	}
	if a.TemplateFile == "" {
		return fmt.Errorf("Missing argument: TemplateFile")
	}
	if a.OutputFile == "" {
		return fmt.Errorf("Missing argument: OutputFile")
	}
	return nil
}

// Groovy returns the agent groovy expression for the agent or `any` if its black
func (a *PipelineAgent) Groovy() string {
	if a.Label != "" {
		return fmt.Sprintf(`{
      label "%s"
    }
`, a.Label)
	}
	// lets use any for Prow
	return "any"
}

// Groovy returns the groovy expression for all of the lifecycles
func (a *PipelineLifecycles) Groovy() string {
	var buffer bytes.Buffer
	for _, l := range []*PipelineLifecycle{a.Setup, a.SetVersion, a.PreBuild, a.Build, a.PostBuild} {
		if l != nil {
			text := l.Groovy()
			buffer.WriteString(text)
		}
	}
	return buffer.String()
}

// Groovy returns the groovy expression for this lifecycle
func (a *PipelineLifecycle) Groovy() string {
	var buffer bytes.Buffer
	for _, s := range a.Steps {
		buffer.WriteString(s.GroovyBlock(indent))
	}
	return buffer.String()
}

// Groovy returns the groovy expression for this step
func (s *PipelineStep) GroovyBlock(parentIndent string) string {
	var buffer bytes.Buffer
	indent := parentIndent
	groovyBlock := false
	if s.Container != "" {
		buffer.WriteString(indent)
		buffer.WriteString("container(\"")
		buffer.WriteString(s.Container)
		buffer.WriteString("\") {\n")
	}
	if s.Comment != "" {
		buffer.WriteString(indent)
		buffer.WriteString("// ")
		buffer.WriteString(s.Comment)
		buffer.WriteString("\n")
	}
	if s.Command != "" {
		buffer.WriteString(indent)
		buffer.WriteString("sh \"")
		buffer.WriteString(s.Command)
		buffer.WriteString("\"\n")
	} else if s.Groovy != "" {
		lines := strings.Split(s.Groovy, "\n")
		lastIdx := len(lines) -1
		for i, line := range lines {
			buffer.WriteString(indent)
			buffer.WriteString(line)
			if i >= lastIdx && len(s.Steps) > 0 {
				groovyBlock = true
				buffer.WriteString(" {")
			}
			buffer.WriteString("\n")
		}
	}
	childIndent := indent + "  "
	for _, child := range s.Steps {
		buffer.WriteString(child.GroovyBlock(childIndent))
	}
	if s.Container != "" || groovyBlock {
		buffer.WriteString(parentIndent)
		buffer.WriteString("}\n")
	}
	return buffer.String()
}

// LoadPipelineConfig returns the pipeline configuration
func LoadPipelineConfig(fileName string) (*PipelineConfig, error) {
	config := PipelineConfig{}
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
func (c *PipelineConfig) IsEmpty() bool {
	empty := &PipelineConfig{}
	return reflect.DeepEqual(empty, c)
}

// SaveConfig saves the configuration file to the given project directory
func (c *PipelineConfig) SaveConfig(fileName string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
}
