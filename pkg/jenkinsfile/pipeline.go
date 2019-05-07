package jenkinsfile

import (
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

// Pipelines contains all the different kinds of pipeline for different branches
type Pipelines struct {
	PullRequest *PipelineLifecycles `json:"pullRequest,omitempty"`
	Release     *PipelineLifecycles `json:"release,omitempty"`
	Feature     *PipelineLifecycles `json:"feature,omitempty"`
	Post        *PipelineLifecycle  `json:"post,omitempty"`
	Overrides   []*PipelineOverride `json:"overrides,omitempty"`
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
	Steps []*syntax.Step `json:"steps,omitempty"`

	// PreSteps if using inheritance then invoke these steps before the base steps
	PreSteps []*syntax.Step `json:"preSteps,omitempty"`

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

// PipelineOverride allows for overriding named steps in the build pack
type PipelineOverride struct {
	Pipelines []string       `json:"pipelines,omitempty"`
	Stages    []string       `json:"stages,omitempty"`
	Name      string         `json:"name"`
	Step      *syntax.Step   `json:"step,omitempty"`
	Steps     []*syntax.Step `json:"steps,omitempty"`
}

// MatchesPipeline returns true if the pipeline name is specified in the override or no pipeline is specified at all in the override
func (p *PipelineOverride) MatchesPipeline(name string) bool {
	if len(p.Pipelines) == 0 {
		return true
	}
	for _, pipeline := range p.Pipelines {
		if pipeline == name {
			return true
		}
	}

	return false
}

// MatchesStage returns true if the stage/lifecycle name is specified in the override or no stage/lifecycle is specified at all in the override
func (p *PipelineOverride) MatchesStage(name string) bool {
	if len(p.Stages) == 0 {
		return true
	}
	for _, stage := range p.Stages {
		if stage == name {
			return true
		}
	}

	return false
}

// PipelineConfig defines the pipeline configuration
type PipelineConfig struct {
	Extends     *PipelineExtends `json:"extends,omitempty"`
	Agent       syntax.Agent     `json:"agent,omitempty"`
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
		if a.SetVersion == nil && lazyCreate {
			a.SetVersion = &PipelineLifecycle{}
		}
		return a.SetVersion, nil
	case "prebuild":
		if a.PreBuild == nil && lazyCreate {
			a.PreBuild = &PipelineLifecycle{}
		}
		return a.PreBuild, nil
	case "build":
		if a.Build == nil && lazyCreate {
			a.Build = &PipelineLifecycle{}
		}
		return a.Build, nil
	case "postbuild":
		if a.PostBuild == nil && lazyCreate {
			a.PostBuild = &PipelineLifecycle{}
		}
		return a.PostBuild, nil
	case "promote":
		if a.Promote == nil && lazyCreate {
			a.Promote = &PipelineLifecycle{}
		}
		return a.Promote, nil
	default:
		return nil, fmt.Errorf("unknown pipeline lifecycle stage: %s", name)
	}
}

// Groovy returns the groovy string for the lifecycles
func (s PipelineLifecycleArray) Groovy() string {
	statements := []*util.Statement{}
	for _, n := range s {
		l := n.Lifecycle
		if l != nil {
			statements = append(statements, l.ToJenkinsfileStatements()...)
		}
	}
	text := util.WriteJenkinsfileStatements(4, statements)
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
func (l *PipelineLifecycle) ToJenkinsfileStatements() []*util.Statement {
	statements := []*util.Statement{}
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
func (l *PipelineLifecycle) CreateStep(mode string, step *syntax.Step) error {
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
		l.Steps = []*syntax.Step{step}
		l.Replace = true
	default:
		return fmt.Errorf("uknown create mode: %s", mode)
	}
	return nil
}

func removeWhenSteps(prow bool, steps []*syntax.Step) []*syntax.Step {
	answer := []*syntax.Step{}
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
	p.PullRequest = ExtendPipelines("pullRequest", p.PullRequest, base.PullRequest, p.Overrides)
	p.Release = ExtendPipelines("release", p.Release, base.Release, p.Overrides)
	p.Feature = ExtendPipelines("feature", p.Feature, base.Feature, p.Overrides)
	p.Post = ExtendLifecycle("", "post", p.Post, base.Post, p.Overrides)
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

func defaultContainerAroundSteps(container string, steps []*syntax.Step) []*syntax.Step {
	if container == "" {
		return steps
	}
	var containerStep *syntax.Step
	result := []*syntax.Step{}
	for _, step := range steps {
		if step.GetImage() != "" {
			result = append(result, step)
		} else {
			if containerStep == nil {
				containerStep = &syntax.Step{
					Image: container,
				}
				result = append(result, containerStep)
			}
			containerStep.Steps = append(containerStep.Steps, step)
		}
	}
	return result
}

func defaultDirAroundSteps(dir string, steps []*syntax.Step) []*syntax.Step {
	if dir == "" {
		return steps
	}
	var dirStep *syntax.Step
	result := []*syntax.Step{}
	for _, step := range steps {
		if step.GetImage() != "" {
			step.Steps = defaultDirAroundSteps(dir, step.Steps)
			result = append(result, step)
		} else if step.Dir != "" {
			result = append(result, step)
		} else {
			if dirStep == nil {
				dirStep = &syntax.Step{
					Dir: dir,
				}
				result = append(result, dirStep)
			}
			dirStep.Steps = append(dirStep.Steps, step)
		}
	}
	return result
}

// LoadPipelineConfig returns the pipeline configuration
func LoadPipelineConfig(fileName string, resolver ImportFileResolver, jenkinsfileRunner bool, clearContainer bool) (*PipelineConfig, error) {
	return LoadPipelineConfigAndMaybeValidate(fileName, resolver, jenkinsfileRunner, clearContainer, true)
}

// LoadPipelineConfigAndMaybeValidate returns the pipeline configuration, optionally after validating the YAML.
func LoadPipelineConfigAndMaybeValidate(fileName string, resolver ImportFileResolver, jenkinsfileRunner bool, clearContainer bool, skipYamlValidation bool) (*PipelineConfig, error) {
	config := PipelineConfig{}
	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		return &config, err
	}
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return &config, errors.Wrapf(err, "Failed to load file %s", fileName)
	}
	if !skipYamlValidation {
		validationErrors, err := util.ValidateYaml(&config, data)
		if err != nil {
			return &config, fmt.Errorf("failed to validate YAML file %s due to %s", fileName, err)
		}
		if len(validationErrors) > 0 {
			return &config, fmt.Errorf("Validation failures in YAML file %s:\n%s", fileName, strings.Join(validationErrors, "\n"))
		}
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
		if c.Agent.GetImage() == "" {
			c.Agent.Image = base.Agent.GetImage()
		} else if base.Agent.GetImage() == "" && c.Agent.GetImage() != "" {
			base.Agent.Image = c.Agent.GetImage()
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
	c.Pipelines.defaultContainerAndDir(c.Agent.GetImage(), c.Agent.Dir)
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
func ExtendPipelines(pipelineName string, parent *PipelineLifecycles, base *PipelineLifecycles, overrides []*PipelineOverride) *PipelineLifecycles {
	if base == nil {
		return parent
	}
	if parent == nil {
		parent = &PipelineLifecycles{}
	}
	return &PipelineLifecycles{
		Setup:      ExtendLifecycle(pipelineName, "setup", parent.Setup, base.Setup, overrides),
		SetVersion: ExtendLifecycle(pipelineName, "setVersion", parent.SetVersion, base.SetVersion, overrides),
		PreBuild:   ExtendLifecycle(pipelineName, "preBuild", parent.PreBuild, base.PreBuild, overrides),
		Build:      ExtendLifecycle(pipelineName, "build", parent.Build, base.Build, overrides),
		PostBuild:  ExtendLifecycle(pipelineName, "postBuild", parent.PostBuild, base.PostBuild, overrides),
		Promote:    ExtendLifecycle(pipelineName, "promote", parent.Promote, base.Promote, overrides),
		// TODO: Actually do extension for Pipeline rather than just copying it wholesale
		Pipeline: parent.Pipeline,
	}
}

// ExtendLifecycle extends the lifecycle with the inherited base lifecycle
func ExtendLifecycle(pipelineName, stageName string, parent *PipelineLifecycle, base *PipelineLifecycle, overrides []*PipelineOverride) *PipelineLifecycle {
	var lifecycle *PipelineLifecycle
	if parent == nil {
		lifecycle = base
	} else if base == nil {
		lifecycle = parent
	} else if parent.Replace {
		lifecycle = parent
	} else {
		steps := []*syntax.Step{}
		steps = append(steps, parent.PreSteps...)
		steps = append(steps, base.Steps...)
		steps = append(steps, parent.Steps...)
		lifecycle = &PipelineLifecycle{
			Steps: steps,
		}
	}

	if lifecycle != nil {
		for _, override := range overrides {
			if override.MatchesPipeline(pipelineName) && override.MatchesStage(stageName) {
				overriddenSteps := []*syntax.Step{}

				for _, s := range lifecycle.Steps {
					overriddenSteps = append(overriddenSteps, overrideStep(s, override)...)
				}

				lifecycle.Steps = overriddenSteps
			}
		}
	}

	return lifecycle
}

func overrideStep(step *syntax.Step, override *PipelineOverride) []*syntax.Step {
	if step.Name == override.Name {
		if override.Step != nil {
			return []*syntax.Step{override.Step}
		}
		if override.Steps != nil {
			return override.Steps
		}
		return []*syntax.Step{}
	}

	if len(step.Steps) > 0 {
		newSteps := []*syntax.Step{}
		for _, s := range step.Steps {
			newSteps = append(newSteps, overrideStep(s, override)...)
		}
		step.Steps = newSteps
	}

	return []*syntax.Step{step}
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
