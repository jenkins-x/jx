package jenkinsfile

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
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

	// the modes of adding a step

	// CreateStepModePre creates steps before any existing steps
	CreateStepModePre = "pre"

	// CreateStepModePost creates steps after the existing steps
	CreateStepModePost = "post"

	// CreateStepModeReplace replaces the existing steps with the new steps
	CreateStepModeReplace = "replace"
)

var (
	// PipelineKinds the possible values of pipeline
	PipelineKinds = []string{PipelineKindRelease, PipelineKindPullRequest, PipelineKindFeature}

	// PipelineLifecycleNames the possible names of lifecycles of pipeline
	PipelineLifecycleNames = []string{"setup", "setversion", "prebuild", "build", "postbuild", "promote"}

	// CreateStepModes the step creation modes
	CreateStepModes = []string{CreateStepModePre, CreateStepModePost, CreateStepModeReplace}
)

// PipelineAgent contains the agent definition metadata
type PipelineAgent struct {
	Label     string `json:"label,omitempty"`
	Container string `json:"container,omitempty"`
	Dir       string `json:"dir,omitempty"`
}

// Pipelines contains all the different kinds of pipeline for different branches
type Pipelines struct {
	PullRequest *PipelineLifecycles `json:"pullRequest,omitempty"`
	Release     *PipelineLifecycles `json:"release,omitempty"`
	Feature     *PipelineLifecycles `json:"feature,omitempty"`
	Post        *PipelineLifecycle  `json:"post,omitempty"`
}

// PipelineStep defines an individual step in a pipeline, either a command (sh) or groovy block
type PipelineStep struct {
	Name      string          `json:"name,omitempty"`
	Comment   string          `json:"comment,omitempty"`
	Container string          `json:"container,omitempty"`
	Dir       string          `json:"dir,omitempty"`
	Command   string          `json:"sh,omitempty"`
	Groovy    string          `json:"groovy,omitempty"`
	Steps     []*PipelineStep `json:"steps,omitempty"`
	When      string          `json:"when,omitempty"`
}

// PipelineLifecycles defines the steps of a lifecycle section
type PipelineLifecycles struct {
	Setup      *PipelineLifecycle     `json:"setup,omitempty"`
	SetVersion *PipelineLifecycle     `json:"setVersion,omitempty"`
	PreBuild   *PipelineLifecycle     `json:"preBuild,omitempty"`
	Build      *PipelineLifecycle     `json:"build,omitempty"`
	PostBuild  *PipelineLifecycle     `json:"postBuild,omitempty"`
	Promote    *PipelineLifecycle     `json:"promote,omitempty"`
	Pipeline   *syntax.ParsedPipeline `json:"pipeline,omitempty"`
}

// PipelineLifecycle defines the steps of a lifecycle section
type PipelineLifecycle struct {
	Steps []*PipelineStep `json:"steps,omitempty"`

	// PreSteps if using inheritance then invoke these steps before the base steps
	PreSteps []*PipelineStep `json:"preSteps,omitempty"`

	// Replace if using inheritance then replace steps from the base pipeline
	Replace bool `json:"replace,omitempty"`
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
	Import string `json:"import,omitempty"`
	File   string `json:"file,omitempty"`
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
	Extends     *PipelineExtends `json:"extends,omitempty"`
	Agent       PipelineAgent    `json:"agent,omitempty"`
	Env         []corev1.EnvVar  `json:"env,omitempty"`
	Environment string           `json:"environment,omitempty"`
	Pipelines   Pipelines        `json:"pipelines,omitempty"`
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

// GetLifecycle returns the pipeline lifecycle of the given name lazy creating on the fly if required
// or returns an error if the name is not valid
func (a *PipelineLifecycles) GetLifecycle(name string, lazyCreate bool) (*PipelineLifecycle, error) {
	switch name {
	case "setup":
		if a.Setup == nil && lazyCreate {
			a.Setup = &PipelineLifecycle{}
		}
		return a.Setup, nil
	case "setversion":
		if a.Setup == nil && lazyCreate {
			a.Setup = &PipelineLifecycle{}
		}
		return a.Setup, nil
	case "prebuild":
		if a.Setup == nil && lazyCreate {
			a.Setup = &PipelineLifecycle{}
		}
		return a.Setup, nil
	case "build":
		if a.Setup == nil && lazyCreate {
			a.Setup = &PipelineLifecycle{}
		}
		return a.Setup, nil
	case "postbuild":
		if a.Setup == nil && lazyCreate {
			a.Setup = &PipelineLifecycle{}
		}
		return a.Setup, nil
	default:
		return nil, fmt.Errorf("unknown pipeline lifecycle stage: %s", name)
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

// PutAllEnvVars puts all the defined environment variables in the given map
func (l *NamedLifecycle) PutAllEnvVars(m map[string]string) {
	if l.Lifecycle != nil {
		for _, step := range l.Lifecycle.Steps {
			step.PutAllEnvVars(m)
		}
	}
}

// Groovy returns the groovy expression for this lifecycle
func (l *PipelineLifecycle) Groovy() string {
	nl := &NamedLifecycle{Name: "", Lifecycle: l}
	return nl.Groovy()
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

// CreateStep creates the given step using the mode
func (l *PipelineLifecycle) CreateStep(mode string, step *PipelineStep) error {
	err := step.Validate()
	if err != nil {
		return err
	}
	switch mode {
	case CreateStepModePre:
		l.PreSteps = append(l.PreSteps, step)
	case CreateStepModePost:
		l.Steps = append(l.Steps, step)
	case CreateStepModeReplace:
		l.Steps = []*PipelineStep{step}
		l.Replace = true
	default:
		return fmt.Errorf("uknown create mode: %s", mode)
	}
	return nil
}

func removeWhenSteps(prow bool, steps []*PipelineStep) []*PipelineStep {
	answer := []*PipelineStep{}
	for _, step := range steps {
		when := strings.TrimSpace(step.When)
		if prow && when == "!prow" {
			continue
		}
		if !prow && when == "prow" {
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

// AllMap returns all the lifecycles in this pipeline indexed by the pipeline name
func (p *Pipelines) AllMap() map[string]*PipelineLifecycles {
	m := map[string]*PipelineLifecycles{}
	if p.PullRequest != nil {
		m[PipelineKindPullRequest] = p.PullRequest
	}
	if p.Feature != nil {
		m[PipelineKindFeature] = p.Feature
	}
	if p.Release != nil {
		m[PipelineKindRelease] = p.Release
	}
	return m
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

// GetPipeline returns the pipeline for the given name, creating if required if lazyCreate is true or returns an error if its not a valid name
func (p *Pipelines) GetPipeline(kind string, lazyCreate bool) (*PipelineLifecycles, error) {
	switch kind {
	case PipelineKindRelease:
		if p.Release == nil && lazyCreate {
			p.Release = &PipelineLifecycles{}
		}
		return p.Release, nil
	case PipelineKindPullRequest:
		if p.PullRequest == nil && lazyCreate {
			p.PullRequest = &PipelineLifecycles{}
		}
		return p.PullRequest, nil
	case PipelineKindFeature:
		if p.Feature == nil && lazyCreate {
			p.Feature = &PipelineLifecycles{}
		}
		return p.Feature, nil
	default:
		return nil, fmt.Errorf("no such pipeline kind: %s", kind)
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

// Validate validates the step is populated correctly
func (s *PipelineStep) Validate() error {
	if len(s.Steps) > 0 || s.Command != "" {
		return nil
	}
	return fmt.Errorf("invalid step %#v as no child steps or command", s)
}

// PutAllEnvVars puts all the defined environment variables in the given map
func (s *PipelineStep) PutAllEnvVars(m map[string]string) {
	for _, step := range s.Steps {
		step.PutAllEnvVars(m)
	}
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

// GetAllEnvVars finds all the environment variables defined in all pipelines + steps with the first value we find
func (c *PipelineConfig) GetAllEnvVars() map[string]string {
	answer := map[string]string{}

	for _, pipeline := range c.Pipelines.All() {
		if pipeline != nil {
			for _, lifecycle := range pipeline.All() {
				lifecycle.PutAllEnvVars(answer)
			}
		}
	}
	for _, env := range c.Env {
		if env.Value != "" || answer[env.Name] == "" {
			answer[env.Name] = env.Value
		}
	}
	return answer

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
		// TODO: Actually do extension for Pipeline rather than just copying it wholesale
		Pipeline: parent.Pipeline,
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
