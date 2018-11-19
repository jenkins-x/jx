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
	Dir       string          `yaml:"dir,omitempty"`
	Command   string          `yaml:"sh,omitempty"`
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

// PipelineLifecycleArray an array of lifecycle pointers
type PipelineLifecycleArray []*PipelineLifecycle

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
  }`, a.Label)
	}
	// lets use any for Prow
	return "any"
}

// Groovy returns the groovy expression for all of the lifecycles
func (a *PipelineLifecycles) Groovy() string {
	lifecycles := PipelineLifecycleArray([]*PipelineLifecycle{a.Setup, a.SetVersion, a.PreBuild, a.Build, a.PostBuild, a.Promote})
	return lifecycles.Groovy()
}

// AllButPromote returns all lifecycles but promote
func (a *PipelineLifecycles) AllButPromote() PipelineLifecycleArray {
	return []*PipelineLifecycle{a.Setup, a.SetVersion, a.PreBuild, a.Build, a.PostBuild}
}

// Groovy returns the groovy string for the lifecycles
func (s PipelineLifecycleArray) Groovy() string {
	statements := []JenkinsfileStatement{}
	for _, l := range s {
		if l != nil {
			statements = append(statements, l.ToJenkinsfileStatements()...)
		}
	}
	text := WriteJenkinsfileStatements(4, statements)
	// lets remove the very last newline so its easier to compose in templates
	text = strings.TrimSuffix(text, "\n")
	return text
}

// Groovy returns the groovy expression for this lifecycle
func (a *PipelineLifecycle) Groovy() string {
	lifecycles := PipelineLifecycleArray([]*PipelineLifecycle{a})
	return lifecycles.Groovy()
}

// ToJenkinsfileStatements converts the lifecycle to one or more jenkinsfile statements
func (l *PipelineLifecycle) ToJenkinsfileStatements() []JenkinsfileStatement {
	statements := []JenkinsfileStatement{}
	for _, step := range l.Steps {
		statements = append(statements, step.ToJenkinsfileStatements()...)
	}
	return statements
}

// Groovy returns the groovy expression for this step
func (s *PipelineStep) GroovyBlock(parentIndent string) string {
	var buffer bytes.Buffer
	indent := parentIndent
	if s.Comment != "" {
		buffer.WriteString(indent)
		buffer.WriteString("// ")
		buffer.WriteString(s.Comment)
		buffer.WriteString("\n")
	}
	if s.Container != "" {
		buffer.WriteString(indent)
		buffer.WriteString("container('")
		buffer.WriteString(s.Container)
		buffer.WriteString("') {\n")
	} else if s.Dir != "" {
		buffer.WriteString(indent)
		buffer.WriteString("dir('")
		buffer.WriteString(s.Dir)
		buffer.WriteString("') {\n")
	} else if s.Command != "" {
		buffer.WriteString(indent)
		buffer.WriteString("sh \"")
		buffer.WriteString(s.Command)
		buffer.WriteString("\"\n")
	} else if s.Groovy != "" {
		lines := strings.Split(s.Groovy, "\n")
		lastIdx := len(lines) - 1
		for i, line := range lines {
			buffer.WriteString(indent)
			buffer.WriteString(line)
			if i >= lastIdx && len(s.Steps) > 0 {
				buffer.WriteString(" {")
			}
			buffer.WriteString("\n")
		}
	}
	childIndent := indent + "  "
	for _, child := range s.Steps {
		buffer.WriteString(child.GroovyBlock(childIndent))
	}
	return buffer.String()
}

// ToJenkinsfileStatements converts the step to one or more jenkinsfile statements
func (s *PipelineStep) ToJenkinsfileStatements() []JenkinsfileStatement {
	statements := []JenkinsfileStatement{}
	if s.Comment != "" {
		statements = append(statements, JenkinsfileStatement{
			Statement: "",
		}, JenkinsfileStatement{
			Statement: "// " + s.Comment,
		})
	}
	if s.Container != "" {
		statements = append(statements, JenkinsfileStatement{
			Function:  "container",
			Arguments: []string{s.Container},
		})
	} else if s.Dir != "" {
		statements = append(statements, JenkinsfileStatement{
			Function:  "dir",
			Arguments: []string{s.Dir},
		})
	} else if s.Command != "" {
		statements = append(statements, JenkinsfileStatement{
			Statement: "sh \"" + s.Command + "\"",
		})
	} else if s.Groovy != "" {
		lines := strings.Split(s.Groovy, "\n")
		for _, line := range lines {
			statements = append(statements, JenkinsfileStatement{
				Statement: line,
			})
		}
	}
	if len(statements) > 0 {
		last := &statements[len(statements)-1]
		for _, c := range s.Steps {
			last.Children = append(last.Children, c.ToJenkinsfileStatements()...)
		}
	}
	return statements
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
