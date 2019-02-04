package jenkinsfile

import (
	"bytes"
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
)

const (
	// PipelineConfigFileName is the name of the pipeline configuration file
	PipelineConfigFileName = "pipeline.yaml"

	// PipelineTemplateFileName defines the jenkisnfile template used to generate the pipeline
	PipelineTemplateFileName = "Jenkinsfile.tmpl"

	// PipelineKindRelease represents a release pipeline triggered on merge to master (or a release branch)
	PipelineKindRelease = "release"

	// PipelineKindPullRequest represents a Pull Request pipeline
	PipelineKindPullRequest = "pullrequest"

	// PipelineKindFeature represents a pipeline on a feature branch
	PipelineKindFeature = "feature"
)

var (
	// PipelineKinds the possible values of pipeline
	PipelineKinds = []string{PipelineKindRelease, PipelineKindPullRequest, PipelineKindFeature}
)

// PipelineAgent contains the agent definition metadata
type PipelineAgent struct {
	Label     string `yaml:"label,omitempty"`
	Container string `yaml:"container,omitempty"`
	Dir       string `yaml:"dir,omitempty"`
}

// Pipelines contains all the different kinds of pipeline for diferent branches
type Pipelines struct {
	PullRequest *PipelineLifecycles `yaml:"pullRequest,omitempty"`
	Release     *PipelineLifecycles `yaml:"release,omitempty"`
	Feature     *PipelineLifecycles `yaml:"feature,omitempty"`
	Post        *PipelineLifecycle  `yaml:"post,omitempty"`
}

// PipelineStep defines an individual step in a pipeline, either a command (sh) or groovy block
type PipelineStep struct {
	Name      string          `yaml:"name,omitempty"`
	Comment   string          `yaml:"comment,omitempty"`
	Container string          `yaml:"container,omitempty"`
	Dir       string          `yaml:"dir,omitempty"`
	Command   string          `yaml:"sh,omitempty"`
	Groovy    string          `yaml:"groovy,omitempty"`
	Steps     []*PipelineStep `yaml:"steps,omitempty"`
	When      string          `yaml:"when,omitempty"`
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

	// PreSteps if using inheritance then invoke these steps before the base steps 
	PreSteps []*PipelineStep `yaml:"preSteps,omitempty"`

	// Replace if using inheritence then replace steps from the base pipeline
	Replace bool `yaml:"replace,omitempty"`
}

// NamedLifecycle a lifecycle and its name
type NamedLifecycle struct {
	Name      string
	Lifecycle *PipelineLifecycle
}

// PipelineLifecycleArray an array of named lifecycle pointers
type PipelineLifecycleArray []NamedLifecycle

// PipelineExtends defines the extension (e.g. parent pipeline which is overloaded
type PipelineExtends struct {
	Import string `yaml:"import,omitempty"`
	File   string `yaml:"file,omitempty"`
}

// ImportFile returns an ImportFile for the given extension
func (x *PipelineExtends) ImportFile() *ImportFile {
	return &ImportFile{
		Import: x.Import,
		File:   x.File,
	}
}

// PipelineConfig defines the pipeline configuration
type PipelineConfig struct {
	Extends     *PipelineExtends `yaml:"extends,omitempty"`
	Agent       PipelineAgent    `yaml:"agent,omitempty"`
	Environment string           `yaml:"environment,omitempty"`
	Pipelines   Pipelines        `yaml:"pipelines,omitempty"`
}

// CreateJenkinsfileArguments contains the arguents to generate a Jenkinsfiles dynamically
type CreateJenkinsfileArguments struct {
	ConfigFile          string
	TemplateFile        string
	OutputFile          string
	JenkinsfileRunner   bool
	ClearContainerNames bool
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
	return a.All().Groovy()
}

// All returns all lifecycles in order
func (a *PipelineLifecycles) All() PipelineLifecycleArray {
	return []NamedLifecycle{
		{"setup", a.Setup},
		{"setversion", a.SetVersion},
		{"prebuild", a.PreBuild},
		{"build", a.Build},
		{"postbuild", a.PostBuild},
		{"promote", a.Promote},
	}
}

// AllButPromote returns all lifecycles but promote
func (a *PipelineLifecycles) AllButPromote() PipelineLifecycleArray {
	return []NamedLifecycle{
		{"setup", a.Setup},
		{"setversion", a.SetVersion},
		{"prebuild", a.PreBuild},
		{"build", a.Build},
		{"postbuild", a.PostBuild},
	}
}

// RemoveWhenStatements removes any when conditions
func (a *PipelineLifecycles) RemoveWhenStatements(prow bool) {
	for _, n := range a.All() {
		v := n.Lifecycle
		if v != nil {
			v.RemoveWhenStatements(prow)
		}
	}
}

// Groovy returns the groovy string for the lifecycles
func (s PipelineLifecycleArray) Groovy() string {
	statements := []*Statement{}
	for _, n := range s {
		l := n.Lifecycle
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
func (l *NamedLifecycle) Groovy() string {
	lifecycles := PipelineLifecycleArray([]NamedLifecycle{*l})
	return lifecycles.Groovy()
}

// ToJenkinsfileStatements converts the lifecycle to one or more jenkinsfile statements
func (l *PipelineLifecycle) ToJenkinsfileStatements() []*Statement {
	statements := []*Statement{}
	for _, step := range l.Steps {
		statements = append(statements, step.ToJenkinsfileStatements()...)
	}
	return statements
}

// RemoveWhenStatements removes any when conditions
func (l *PipelineLifecycle) RemoveWhenStatements(prow bool) {
	l.PreSteps = removeWhenSteps(prow, l.PreSteps)
	l.Steps = removeWhenSteps(prow, l.Steps)
}

func removeWhenSteps(prow bool, steps []*PipelineStep) []*PipelineStep {
	answer := []*PipelineStep{}
	for _, step := range steps {
		when := strings.TrimSpace(step.When)
		if (prow && when == "!prow") {
			continue
		}
		if (!prow && when == "prow") {
			continue
		}
		step.Steps = removeWhenSteps(prow, step.Steps)
		answer = append(answer, step)
	}
	return answer
}

// Extend extends these pipelines with the base pipeline
func (p *Pipelines) Extend(base *Pipelines) error {
	p.PullRequest = ExtendPipelines(p.PullRequest, base.PullRequest)
	p.Release = ExtendPipelines(p.Release, base.Release)
	p.Feature = ExtendPipelines(p.Feature, base.Feature)
	p.Post = ExtendLifecycle(p.Post, base.Post)
	return nil
}

// All returns all the lifecycles in this pipeline, some may be null
func (p *Pipelines) All() []*PipelineLifecycles {
	return []*PipelineLifecycles{p.PullRequest, p.Feature, p.Release}
}

// defaultContainerAndDir defaults the container if none is being used
func (p *Pipelines) defaultContainerAndDir(container string, dir string) {
	defaultContainerAndDir(container, dir, p.All()...)
}

// RemoveWhenStatements removes any prow or !prow statements
func (p *Pipelines) RemoveWhenStatements(prow bool) {
	for _, l := range p.All() {
		if l != nil {
			l.RemoveWhenStatements(prow)
		}
	}
	if p.Post != nil {
		p.Post.RemoveWhenStatements(prow)
	}
}

func defaultContainerAndDir(container string, dir string, lifecycles ...*PipelineLifecycles) {
	for _, l := range lifecycles {
		if l != nil {
			defaultLifecycleContainerAndDir(container, dir, l.All())
		}
	}
}

func defaultLifecycleContainerAndDir(container string, dir string, lifecycles PipelineLifecycleArray) {
	if container == "" && dir == "" {
		return
	}
	for _, n := range lifecycles {
		l := n.Lifecycle
		if l != nil {
			if dir != "" {
				l.PreSteps = defaultDirAroundSteps(dir, l.PreSteps)
				l.Steps = defaultDirAroundSteps(dir, l.Steps)
			}
			if container != "" {
				l.PreSteps = defaultContainerAroundSteps(container, l.PreSteps)
				l.Steps = defaultContainerAroundSteps(container, l.Steps)
			}
		}
	}
}

func defaultContainerAroundSteps(container string, steps []*PipelineStep) []*PipelineStep {
	if container == "" {
		return steps
	}
	var containerStep *PipelineStep
	result := []*PipelineStep{}
	for _, step := range steps {
		if step.Container != "" {
			result = append(result, step)
		} else {
			if containerStep == nil {
				containerStep = &PipelineStep{
					Container: container,
				}
				result = append(result, containerStep)
			}
			containerStep.Steps = append(containerStep.Steps, step)
		}
	}
	return result
}

func defaultDirAroundSteps(dir string, steps []*PipelineStep) []*PipelineStep {
	if dir == "" {
		return steps
	}
	var dirStep *PipelineStep
	result := []*PipelineStep{}
	for _, step := range steps {
		if step.Container != "" {
			step.Steps = defaultDirAroundSteps(dir, step.Steps)
			result = append(result, step)
		} else if step.Dir != "" {
			result = append(result, step)
		} else {
			if dirStep == nil {
				dirStep = &PipelineStep{
					Dir: dir,
				}
				result = append(result, dirStep)
			}
			dirStep.Steps = append(dirStep.Steps, step)
		}
	}
	return result
}

// GroovyBlock returns the groovy expression for this step
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
func (s *PipelineStep) ToJenkinsfileStatements() []*Statement {
	statements := []*Statement{}
	if s.Comment != "" {
		statements = append(statements, &Statement{
			Statement: "",
		}, &Statement{
			Statement: "// " + s.Comment,
		})
	}
	if s.Container != "" {
		statements = append(statements, &Statement{
			Function:  "container",
			Arguments: []string{s.Container},
		})
	} else if s.Dir != "" {
		statements = append(statements, &Statement{
			Function:  "dir",
			Arguments: []string{s.Dir},
		})
	} else if s.Command != "" {
		statements = append(statements, &Statement{
			Statement: "sh \"" + s.Command + "\"",
		})
	} else if s.Groovy != "" {
		lines := strings.Split(s.Groovy, "\n")
		for _, line := range lines {
			statements = append(statements, &Statement{
				Statement: line,
			})
		}
	}
	if len(statements) > 0 {
		last := statements[len(statements)-1]
		for _, c := range s.Steps {
			last.Children = append(last.Children, c.ToJenkinsfileStatements()...)
		}
	}
	return statements
}

// LoadPipelineConfig returns the pipeline configuration
func LoadPipelineConfig(fileName string, resolver ImportFileResolver, jenkinsfileRunner bool, clearContainer bool) (*PipelineConfig, error) {
	config := PipelineConfig{}
	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		return &config, err
	}
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return &config, errors.Wrapf(err, "Failed to load file %s", fileName)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return &config, errors.Wrapf(err, "Failed to unmarshal file %s", fileName)
	}
	pipelines := &config.Pipelines
	pipelines.RemoveWhenStatements(jenkinsfileRunner)
	if clearContainer {
		// lets force any agent for prow / jenkinsfile runner
		config.Agent.Label = ""
		config.Agent.Container = ""
	}
	if config.Extends == nil || config.Extends.File == "" {
		config.defaultContainerAndDir()
		return &config, nil
	}
	file := config.Extends.File
	importModule := config.Extends.Import
	if importModule != "" {
		file, err = resolver(config.Extends.ImportFile())
		if err != nil {
			return &config, errors.Wrapf(err, "Failed to resolve imports for file %s", fileName)
		}

	} else if !filepath.IsAbs(file) {
		dir, _ := filepath.Split(fileName)
		if dir != "" {
			file = filepath.Join(dir, file)
		}
	}
	exists, err = util.FileExists(file)
	if err != nil {
		return &config, errors.Wrapf(err, "base pipeline file does not exist %s", file)
	}
	if !exists {
		return &config, fmt.Errorf("base pipeline file does not exist %s", file)
	}
	basePipeline, err := LoadPipelineConfig(file, resolver, jenkinsfileRunner, clearContainer)
	if err != nil {
		return &config, errors.Wrapf(err, "Failed to base pipeline file %s", file)
	}
	err = config.ExtendPipeline(basePipeline, clearContainer)
	return &config, err
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

// ExtendPipeline inherits this pipeline from the given base pipeline
func (c *PipelineConfig) ExtendPipeline(base *PipelineConfig, clearContainer bool) error {
	if clearContainer {
		c.Agent.Container = ""
		c.Agent.Label = ""
		base.Agent.Container = ""
		base.Agent.Label = ""
	} else {
		if c.Agent.Label == "" {
			c.Agent.Label = base.Agent.Label
		} else if base.Agent.Label == "" && c.Agent.Label != "" {
			base.Agent.Label = c.Agent.Label
		}
		if c.Agent.Container == "" {
			c.Agent.Container = base.Agent.Container
		} else if base.Agent.Container == "" && c.Agent.Container != "" {
			base.Agent.Container = c.Agent.Container
		}
	}
	if c.Agent.Dir == "" {
		c.Agent.Dir = base.Agent.Dir
	} else if base.Agent.Dir == "" && c.Agent.Dir != "" {
		base.Agent.Dir = c.Agent.Dir
	}
	base.defaultContainerAndDir()
	c.defaultContainerAndDir()
	c.Pipelines.Extend(&base.Pipelines)
	return nil
}

func (c *PipelineConfig) defaultContainerAndDir() {
	c.Pipelines.defaultContainerAndDir(c.Agent.Container, c.Agent.Dir)
}

// ExtendPipelines extends the parent lifecycle with the base
func ExtendPipelines(parent *PipelineLifecycles, base *PipelineLifecycles) *PipelineLifecycles {
	if parent == nil {
		return base
	}
	if base == nil {
		return parent
	}
	return &PipelineLifecycles{
		Setup:      ExtendLifecycle(parent.Setup, base.Setup),
		SetVersion: ExtendLifecycle(parent.SetVersion, base.SetVersion),
		PreBuild:   ExtendLifecycle(parent.PreBuild, base.PreBuild),
		Build:      ExtendLifecycle(parent.Build, base.Build),
		PostBuild:  ExtendLifecycle(parent.PostBuild, base.PostBuild),
		Promote:    ExtendLifecycle(parent.Promote, base.Promote),
	}
}

// ExtendLifecycle extends the lifecycle with the inherited base lifecycle
func ExtendLifecycle(parent *PipelineLifecycle, base *PipelineLifecycle) *PipelineLifecycle {
	if parent == nil {
		return base
	}
	if base == nil {
		return parent
	}
	if parent.Replace {
		return parent
	}
	steps := []*PipelineStep{}
	steps = append(steps, parent.PreSteps...)
	steps = append(steps, base.Steps...)
	steps = append(steps, parent.Steps...)
	return &PipelineLifecycle{
		Steps: steps,
	}
}

// GenerateJenkinsfile generates the jenkinsfile
func (a *CreateJenkinsfileArguments) GenerateJenkinsfile(resolver ImportFileResolver) error {
	err := a.Validate()
	if err != nil {
		return err
	}
	config, err := LoadPipelineConfig(a.ConfigFile, resolver, a.JenkinsfileRunner, a.ClearContainerNames)
	if err != nil {
		return err
	}

	templateFile := a.TemplateFile

	data, err := ioutil.ReadFile(templateFile)
	if err != nil {
		return errors.Wrapf(err, "failed to load template %s", templateFile)
	}

	t, err := template.New("myJenkinsfile").Parse(string(data))
	if err != nil {
		return errors.Wrapf(err, "failed to parse template %s", templateFile)
	}
	outFile := a.OutputFile
	outDir, _ := filepath.Split(outFile)
	if outDir != "" {
		err = os.MkdirAll(outDir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to make directory %s when creating Jenkinsfile %s", outDir, outFile)
		}
	}
	file, err := os.Create(outFile)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", outFile)
	}
	defer file.Close()

	err = t.Execute(file, config)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", outFile)
	}
	return nil
}
