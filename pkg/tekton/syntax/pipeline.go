package syntax

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/knative/pkg/apis"
	"github.com/pkg/errors"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// GitMergeImage is the default image name that is used in the git merge step of a pipeline
const GitMergeImage = "rawlingsj/builder-jx:wip34"

// ParsedPipeline is the internal representation of the Pipeline, used to validate and create CRDs
type ParsedPipeline struct {
	Agent   Agent       `json:"agent,omitempty"`
	Env     []EnvVar    `json:"env,omitempty"`
	Options RootOptions `json:"options,omitempty"`
	Stages  []Stage     `json:"stages"`
	Post    []Post      `json:"post,omitempty"`

	// Replaced by Env, retained for backwards compatibility
	Environment []EnvVar `json:"environment,omitempty"`
}

// Agent defines where the pipeline, stage, or step should run.
type Agent struct {
	// One of label or image is required.
	Label string `json:"label,omitempty"`
	Image string `json:"image,omitempty"`

	// Legacy fields from jenkinsfile.PipelineAgent
	Container string `json:"container,omitempty"`
	Dir       string `json:"dir,omitempty"`
}

// EnvVar is a key/value pair defining an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TimeoutUnit is used for calculating timeout duration
type TimeoutUnit string

// The available time units.
const (
	TimeoutUnitSeconds TimeoutUnit = "seconds"
	TimeoutUnitMinutes TimeoutUnit = "minutes"
	TimeoutUnitHours   TimeoutUnit = "hours"
	TimeoutUnitDays    TimeoutUnit = "days"
)

// All possible time units, used for validation
var allTimeoutUnits = []TimeoutUnit{TimeoutUnitSeconds, TimeoutUnitMinutes, TimeoutUnitHours, TimeoutUnitDays}

func allTimeoutUnitsAsStrings() []string {
	tu := make([]string, len(allTimeoutUnits))

	for i, u := range allTimeoutUnits {
		tu[i] = string(u)
	}

	return tu
}

// Timeout defines how long a stage or pipeline can run before timing out.
type Timeout struct {
	Time int64 `json:"time"`
	// Has some sane default - probably seconds
	Unit TimeoutUnit `json:"unit,omitempty"`
}

func (t Timeout) toDuration() (*metav1.Duration, error) {
	durationStr := ""
	// TODO: Populate a default timeout unit, most likely seconds.
	if t.Unit != "" {
		durationStr = fmt.Sprintf("%d%c", t.Time, t.Unit[0])
	} else {
		durationStr = fmt.Sprintf("%ds", t.Time)
	}

	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, err
	}
	return &metav1.Duration{Duration: d}, nil
}

// RootOptions contains options that can be configured on either a pipeline or a stage
type RootOptions struct {
	Timeout Timeout `json:"timeout,omitempty"`
	// TODO: Not yet implemented in build-pipeline
	Retry int8 `json:"retry,omitempty"`
	// ContainerOptions allows for advanced configuration of containers for a single stage or the whole
	// pipeline, adding to configuration that can be configured through the syntax already. This includes things
	// like CPU/RAM requests/limits, secrets, ports, etc. Some of these things will end up with native syntax approaches
	// down the road.
	ContainerOptions *corev1.Container `json:"containerOptions,omitempty"`
}

// Stash defines files to be saved for use in a later stage, marked with a name
type Stash struct {
	Name string `json:"name"`
	// Eventually make this optional so that you can do volumes instead
	Files string `json:"files"`
}

// Unstash defines a previously-defined stash to be copied into this stage's workspace
type Unstash struct {
	Name string `json:"name"`
	Dir  string `json:"dir,omitempty"`
}

// StageOptions contains both options that can be configured on either a pipeline or a stage, via
// RootOptions, or stage-specific options.
type StageOptions struct {
	RootOptions `json:",inline"`

	// TODO: Not yet implemented in build-pipeline
	Stash   Stash   `json:"stash,omitempty"`
	Unstash Unstash `json:"unstash,omitempty"`

	Workspace *string `json:"workspace,omitempty"`
}

// Step defines a single step, from the author's perspective, to be executed within a stage.
type Step struct {
	// An optional name to give the step for reporting purposes
	Name string `json:"name,omitempty"`

	// One of command, step, or loop is required.
	Command string `json:"command,omitempty"`
	// args is optional, but only allowed with command
	Arguments []string `json:"args,omitempty"`
	// dir is optional, but only allowed with command. Refers to subdirectory of workspace
	Dir string `json:"dir,omitempty"`

	Step string `json:"step,omitempty"`
	// options is optional, but only allowed with step
	// Also, we'll need to do some magic to do type verification during translation - i.e., this step wants a number
	// for this option, so translate the string value for that option to a number.
	Options map[string]string `json:"options,omitempty"`

	Loop *Loop `json:"loop,omitempty"`

	// agent can be overridden on a step
	Agent *Agent `json:"agent,omitempty"`

	// Image alows the docker image for a step to be specified
	Image string `json:"image,omitempty"`

	// env allows defining per-step environment variables
	Env []EnvVar `json:"env,omitempty"`

	// Legacy fields from jenkinsfile.PipelineStep before it was eliminated.
	Comment   string  `json:"comment,omitempty"`
	Groovy    string  `json:"groovy,omitempty"`
	Steps     []*Step `json:"steps,omitempty"`
	When      string  `json:"when,omitempty"`
	Container string  `json:"container,omitempty"`
	Sh        string  `json:"sh,omitempty"`
}

// Loop is a special step that defines a variable, a list of possible values for that variable, and a set of steps to
// repeat for each value for the variable, with the variable set with that value in the environment for the execution of
// those steps.
type Loop struct {
	// The variable name.
	Variable string `json:"variable"`
	// The list of values to iterate over
	Values []string `json:"values"`
	// The steps to run
	Steps []Step `json:"steps"`
}

// Stage is a unit of work in a pipeline, corresponding either to a Task or a set of Tasks to be run sequentially or in
// parallel with common configuration.
type Stage struct {
	Name     string       `json:"name"`
	Agent    Agent        `json:"agent,omitempty"`
	Env      []EnvVar     `json:"env,omitempty"`
	Options  StageOptions `json:"options,omitempty"`
	Steps    []Step       `json:"steps,omitempty"`
	Stages   []Stage      `json:"stages,omitempty"`
	Parallel []Stage      `json:"parallel,omitempty"`
	Post     []Post       `json:"post,omitempty"`

	// Replaced by Env, retained for backwards compatibility
	Environment []EnvVar `json:"environment,omitempty"`
}

// PostCondition is used to specify under what condition a post action should be executed.
type PostCondition string

// Probably extensible down the road
const (
	PostConditionSuccess PostCondition = "success"
	PostConditionFailure PostCondition = "failure"
	PostConditionAlways  PostCondition = "always"
)

// Post contains a PostCondition and one more actions to be executed after a pipeline or stage if the condition is met.
type Post struct {
	// TODO: Conditional execution of something after a Task or Pipeline completes is not yet implemented
	Condition PostCondition `json:"condition"`
	Actions   []PostAction  `json:"actions"`
}

// PostAction contains the name of a built-in post action and options to pass to that action.
type PostAction struct {
	// TODO: Notifications are not yet supported in Build Pipeline per se.
	Name string `json:"name"`
	// Also, we'll need to do some magic to do type verification during translation - i.e., this action wants a number
	// for this option, so translate the string value for that option to a number.
	Options map[string]string `json:"options,omitempty"`
}

var _ apis.Validatable = (*ParsedPipeline)(nil)

func (s *Stage) taskName() string {
	return strings.ToLower(strings.NewReplacer(" ", "-").Replace(s.Name))
}

// stageLabelName replaces invalid characters in stage names for label usage.
func (s *Stage) stageLabelName() string {
	return MangleToRfc1035Label(s.Name, "")
}

// GroovyBlock returns the groovy expression for this step
// Legacy code for Jenkinsfile generation
func (s *Step) GroovyBlock(parentIndent string) string {
	var buffer bytes.Buffer
	indent := parentIndent
	if s.Comment != "" {
		buffer.WriteString(indent)
		buffer.WriteString("// ")
		buffer.WriteString(s.Comment)
		buffer.WriteString("\n")
	}
	if s.GetImage() != "" {
		buffer.WriteString(indent)
		buffer.WriteString("container('")
		buffer.WriteString(s.GetImage())
		buffer.WriteString("') {\n")
	} else if s.Dir != "" {
		buffer.WriteString(indent)
		buffer.WriteString("dir('")
		buffer.WriteString(s.Dir)
		buffer.WriteString("') {\n")
	} else if s.GetFullCommand() != "" {
		buffer.WriteString(indent)
		buffer.WriteString("sh \"")
		buffer.WriteString(s.GetFullCommand())
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
// Legacy code for Jenkinsfile generation
func (s *Step) ToJenkinsfileStatements() []*util.Statement {
	statements := []*util.Statement{}
	if s.Comment != "" {
		statements = append(statements, &util.Statement{
			Statement: "",
		}, &util.Statement{
			Statement: "// " + s.Comment,
		})
	}
	if s.GetImage() != "" {
		statements = append(statements, &util.Statement{
			Function:  "container",
			Arguments: []string{s.GetImage()},
		})
	} else if s.Dir != "" {
		statements = append(statements, &util.Statement{
			Function:  "dir",
			Arguments: []string{s.Dir},
		})
	} else if s.GetFullCommand() != "" {
		statements = append(statements, &util.Statement{
			Statement: "sh \"" + s.GetFullCommand() + "\"",
		})
	} else if s.Groovy != "" {
		lines := strings.Split(s.Groovy, "\n")
		for _, line := range lines {
			statements = append(statements, &util.Statement{
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
// Legacy code for Jenkinsfile generation
func (s *Step) Validate() error {
	if len(s.Steps) > 0 || s.GetCommand() != "" {
		return nil
	}
	return fmt.Errorf("invalid step %#v as no child steps or command", s)
}

// PutAllEnvVars puts all the defined environment variables in the given map
// Legacy code for Jenkinsfile generation
func (s *Step) PutAllEnvVars(m map[string]string) {
	for _, step := range s.Steps {
		step.PutAllEnvVars(m)
	}
}

// GetCommand gets the step's command to execute, opting for Command if set, then Sh.
func (s *Step) GetCommand() string {
	if s.Command != "" {
		return s.Command
	}

	return s.Sh
}

// GetFullCommand gets the full command to execute, including arguments.
func (s *Step) GetFullCommand() string {
	cmd := s.GetCommand()

	// If GetCommand() was an empty string, don't deal with arguments, just return.
	if len(s.Arguments) > 0 && cmd != "" {
		cmd = fmt.Sprintf("%s %s", cmd, strings.Join(s.Arguments, " "))
	}

	return cmd
}

// GetImage gets the step's image to run on, opting for Image if set, then Container.
func (s *Step) GetImage() string {
	if s.Image != "" {
		return s.Image
	}
	if s.Agent != nil && s.Agent.Image != "" {
		return s.Agent.Image
	}

	return s.Container
}

// Groovy returns the agent groovy expression for the agent or `any` if its blank
// Legacy code for Jenkinsfile generation
func (a *Agent) Groovy() string {
	if a.Label != "" {
		return fmt.Sprintf(`{
    label "%s"
  }`, a.Label)
	}
	// lets use any for Prow
	return "any"
}

// GetImage gets the agent's image to run on, opting for Image if set, then Container.
func (a *Agent) GetImage() string {
	if a.Image != "" {
		return a.Image
	}

	return a.Container
}

// MangleToRfc1035Label - Task/Step names need to be RFC 1035/1123 compliant DNS labels, so we mangle
// them to make them compliant. Results should match the following regex and be
// no more than 63 characters long:
//     [a-z]([-a-z0-9]*[a-z0-9])?
// cf. https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
// body is assumed to have at least one ASCII letter.
// suffix is assumed to be alphanumeric and non-empty.
// TODO: Combine with kube.ToValidName (that function needs to handle lengths)
func MangleToRfc1035Label(body string, suffix string) string {
	const maxLabelLength = 63
	maxBodyLength := maxLabelLength - len(suffix) - 1 // Add an extra hyphen before the suffix

	var sb strings.Builder
	bufferedHyphen := false // Used to make sure we don't output consecutive hyphens.
	for _, codepoint := range body {
		toWrite := 0
		if sb.Len() != 0 { // Digits and hyphens aren't allowed to be the first character
			if codepoint == ' ' || codepoint == '-' || codepoint == '.' {
				bufferedHyphen = true
			} else if codepoint >= '0' && codepoint <= '9' {
				toWrite = 1
			}
		}

		if codepoint >= 'A' && codepoint <= 'Z' {
			codepoint += ('a' - 'A') // Offset to make character lowercase
			toWrite = 1
		} else if codepoint >= 'a' && codepoint <= 'z' {
			toWrite = 1
		}

		if toWrite > 0 {
			if bufferedHyphen {
				toWrite++
			}
			if sb.Len()+toWrite > maxBodyLength {
				break
			}
			if bufferedHyphen {
				sb.WriteRune('-')
				bufferedHyphen = false
			}
			sb.WriteRune(codepoint)
		}
	}

	if suffix != "" {
		sb.WriteRune('-')
		sb.WriteString(suffix)
	}
	return sb.String()
}

// GetEnv gets the environment for the ParsedPipeline, returning Env first and Environment if Env isn't populated.
func (j *ParsedPipeline) GetEnv() []EnvVar {
	if len(j.Env) > 0 {
		return j.Env
	}

	return j.Environment
}

// GetEnv gets the environment for the Stage, returning Env first and Environment if Env isn't populated.
func (s *Stage) GetEnv() []EnvVar {
	if len(s.Env) > 0 {
		return s.Env
	}

	return s.Environment
}

// Validate checks the parsed ParsedPipeline to find any errors in it.
// TODO: Improve validation to actually return all the errors via the nested errors?
// TODO: Add validation for the not-yet-supported-for-CRD-generation sections
func (j *ParsedPipeline) Validate(context context.Context) *apis.FieldError {
	if err := validateAgent(j.Agent).ViaField("agent"); err != nil {
		return err
	}

	if err := validateStages(j.Stages, j.Agent); err != nil {
		return err
	}

	if err := validateStageNames(j); err != nil {
		return err
	}

	if err := validateRootOptions(j.Options).ViaField("options"); err != nil {
		return err
	}

	return nil
}

func validateAgent(a Agent) *apis.FieldError {
	// TODO: This is the same whether you specify an agent without label or image, or if you don't specify an agent
	// at all, which is nonoptimal.
	if !equality.Semantic.DeepEqual(a, Agent{}) {
		if a.Container != "" {
			return &apis.FieldError{
				Message: "the container field is deprecated - please use image instead",
				Paths:   []string{"container"},
			}
		}
		if a.Dir != "" {
			return &apis.FieldError{
				Message: "the dir field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it.",
				Paths:   []string{"dir"},
			}
		}

		if a.Image != "" && a.Label != "" {
			return apis.ErrMultipleOneOf("label", "image")
		}

		if a.Image == "" && a.Label == "" {
			return apis.ErrMissingOneOf("label", "image")
		}
	}

	return nil
}

var containsASCIILetter = regexp.MustCompile(`[a-zA-Z]`).MatchString

func validateStage(s Stage, parentAgent Agent) *apis.FieldError {
	if len(s.Steps) == 0 && len(s.Stages) == 0 && len(s.Parallel) == 0 {
		return apis.ErrMissingOneOf("steps", "stages", "parallel")
	}

	if !containsASCIILetter(s.Name) {
		return &apis.FieldError{
			Message: "Stage name must contain at least one ASCII letter",
			Paths:   []string{"name"},
		}
	}

	stageAgent := s.Agent
	if equality.Semantic.DeepEqual(stageAgent, Agent{}) {
		stageAgent = parentAgent
	}

	if equality.Semantic.DeepEqual(stageAgent, Agent{}) {
		return &apis.FieldError{
			Message: "No agent specified for stage or for its parent(s)",
			Paths:   []string{"agent"},
		}
	}

	if len(s.Steps) > 0 {
		if len(s.Stages) > 0 || len(s.Parallel) > 0 {
			return apis.ErrMultipleOneOf("steps", "stages", "parallel")
		}
		seenStepNames := make(map[string]int)
		for i, step := range s.Steps {
			if err := validateStep(step).ViaFieldIndex("steps", i); err != nil {
				return err
			}
			if step.Name != "" {
				if count, exists := seenStepNames[step.Name]; exists {
					seenStepNames[step.Name] = count + 1
				} else {
					seenStepNames[step.Name] = 1
				}
			}
		}

		var duplicateSteps []string
		for k, v := range seenStepNames {
			if v > 1 {
				duplicateSteps = append(duplicateSteps, k)
			}
		}
		if len(duplicateSteps) > 0 {
			sort.Strings(duplicateSteps)
			return &apis.FieldError{
				Message: "step names within a stage must be unique",
				Details: fmt.Sprintf("The following step names in the stage %s are used more than once: %s", s.Name, strings.Join(duplicateSteps, ", ")),
				Paths:   []string{"steps"},
			}
		}
	}

	if len(s.Stages) > 0 {
		if len(s.Parallel) > 0 {
			return apis.ErrMultipleOneOf("steps", "stages", "parallel")
		}
		for i, stage := range s.Stages {
			if err := validateStage(stage, parentAgent).ViaFieldIndex("stages", i); err != nil {
				return err
			}
		}
	}

	if len(s.Parallel) > 0 {
		for i, stage := range s.Parallel {
			if err := validateStage(stage, parentAgent).ViaFieldIndex("parallel", i); err != nil {
				return nil
			}
		}
	}

	return validateStageOptions(s.Options).ViaField("options")
}

func moreThanOneAreTrue(vals ...bool) bool {
	count := 0

	for _, v := range vals {
		if v {
			count++
		}
	}

	return count > 1
}

func validateStep(s Step) *apis.FieldError {
	// Special cases for when you use legacy build pack syntax inside a pipeline definition
	if s.Sh != "" {
		return &apis.FieldError{
			Message: "the sh field is deprecated - please use command instead",
			Paths:   []string{"sh"},
		}
	}
	if s.Container != "" {
		return &apis.FieldError{
			Message: "the container field is deprecated - please use image instead",
			Paths:   []string{"container"},
		}
	}
	if s.Groovy != "" {
		return &apis.FieldError{
			Message: "the groovy field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it.",
			Paths:   []string{"groovy"},
		}
	}
	if s.Comment != "" {
		return &apis.FieldError{
			Message: "the comment field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it.",
			Paths:   []string{"comment"},
		}
	}
	if s.When != "" {
		return &apis.FieldError{
			Message: "the when field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it.",
			Paths:   []string{"when"},
		}
	}
	if len(s.Steps) > 0 {
		return &apis.FieldError{
			Message: "the steps field is only valid in legacy build packs, not in jenkins-x.yml. Please remove it and list the nested stages sequentially instead.",
			Paths:   []string{"steps"},
		}
	}

	if s.Command == "" && s.Step == "" && s.Loop == nil {
		return apis.ErrMissingOneOf("command", "step", "loop")
	}

	if moreThanOneAreTrue(s.Command != "", s.Step != "", s.Loop != nil) {
		return apis.ErrMultipleOneOf("command", "step", "loop")
	}

	if (s.Command != "" || s.Loop != nil) && len(s.Options) != 0 {
		return &apis.FieldError{
			Message: "Cannot set options for a command or a loop",
			Paths:   []string{"options"},
		}
	}

	if (s.Step != "" || s.Loop != nil) && len(s.Arguments) != 0 {
		return &apis.FieldError{
			Message: "Cannot set command-line arguments for a step or a loop",
			Paths:   []string{"args"},
		}
	}

	if err := validateLoop(s.Loop); err != nil {
		return err.ViaField("loop")
	}

	if s.Agent != nil {
		return validateAgent(*s.Agent).ViaField("agent")
	}
	return nil
}

func validateLoop(l *Loop) *apis.FieldError {
	if l != nil {
		if l.Variable == "" {
			return apis.ErrMissingField("variable")
		}

		if len(l.Steps) == 0 {
			return apis.ErrMissingField("steps")
		}

		if len(l.Values) == 0 {
			return apis.ErrMissingField("values")
		}

		for i, step := range l.Steps {
			if err := validateStep(step).ViaFieldIndex("steps", i); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateStages(stages []Stage, parentAgent Agent) *apis.FieldError {
	if len(stages) == 0 {
		return apis.ErrMissingField("stages")
	}

	for i, s := range stages {
		if err := validateStage(s, parentAgent).ViaFieldIndex("stages", i); err != nil {
			return err
		}
	}

	return nil
}

func validateRootOptions(o RootOptions) *apis.FieldError {
	if !equality.Semantic.DeepEqual(o, RootOptions{}) {
		if !equality.Semantic.DeepEqual(o.Timeout, Timeout{}) {
			if err := validateTimeout(o.Timeout); err != nil {
				return err.ViaField("timeout")
			}
		}

		// TODO: retry will default to 0, so we're kinda stuck checking if it's less than zero here.
		if o.Retry < 0 {
			return &apis.FieldError{
				Message: "Retry count cannot be negative",
				Paths:   []string{"retry"},
			}
		}

		return validateContainerOptions(o.ContainerOptions).ViaField("containerOptions")
	}

	return nil
}

func validateContainerOptions(c *corev1.Container) *apis.FieldError {
	if c != nil {
		if len(c.Command) != 0 {
			return &apis.FieldError{
				Message: "Command cannot be specified in containerOptions",
				Paths:   []string{"command"},
			}
		}
		if len(c.Args) != 0 {
			return &apis.FieldError{
				Message: "Arguments cannot be specified in containerOptions",
				Paths:   []string{"args"},
			}
		}
		if c.Image != "" {
			return &apis.FieldError{
				Message: "Image cannot be specified in containerOptions",
				Paths:   []string{"image"},
			}
		}
		if c.WorkingDir != "" {
			return &apis.FieldError{
				Message: "WorkingDir cannot be specified in containerOptions",
				Paths:   []string{"workingDir"},
			}
		}
		if c.Name != "" {
			return &apis.FieldError{
				Message: "Name cannot be specified in containerOptions",
				Paths:   []string{"name"},
			}
		}
		if c.Stdin {
			return &apis.FieldError{
				Message: "Stdin cannot be specified in containerOptions",
				Paths:   []string{"stdin"},
			}
		}
		if c.TTY {
			return &apis.FieldError{
				Message: "TTY cannot be specified in containerOptions",
				Paths:   []string{"tty"},
			}
		}
	}

	return nil
}

func validateStageOptions(o StageOptions) *apis.FieldError {
	if !equality.Semantic.DeepEqual(o.Stash, Stash{}) {
		if err := validateStash(o.Stash); err != nil {
			return err.ViaField("stash")
		}
	}

	if !equality.Semantic.DeepEqual(o.Unstash, Unstash{}) {
		if err := validateUnstash(o.Unstash); err != nil {
			return err.ViaField("unstash")
		}
	}

	if o.Workspace != nil {
		if err := validateWorkspace(*o.Workspace); err != nil {
			return err
		}
	}

	return validateRootOptions(o.RootOptions)
}

func validateTimeout(t Timeout) *apis.FieldError {
	if !equality.Semantic.DeepEqual(t, Timeout{}) {
		isAllowed := false
		for _, allowed := range allTimeoutUnits {
			if t.Unit == allowed {
				isAllowed = true
			}
		}

		if !isAllowed {
			return &apis.FieldError{
				Message: fmt.Sprintf("%s is not a valid time unit. Valid time units are %s", string(t.Unit),
					strings.Join(allTimeoutUnitsAsStrings(), ", ")),
				Paths: []string{"unit"},
			}
		}

		if t.Time < 1 {
			return &apis.FieldError{
				Message: "Timeout must be greater than zero",
				Paths:   []string{"time"},
			}
		}
	}

	return nil
}

func validateUnstash(u Unstash) *apis.FieldError {
	if !equality.Semantic.DeepEqual(u, Unstash{}) {
		// TODO: Check to make sure the corresponding stash is defined somewhere
		if u.Name == "" {
			return &apis.FieldError{
				Message: "The unstash name must be provided",
				Paths:   []string{"name"},
			}
		}
	}

	return nil
}

func validateStash(s Stash) *apis.FieldError {
	if !equality.Semantic.DeepEqual(s, Stash{}) {
		if s.Name == "" {
			return &apis.FieldError{
				Message: "The stash name must be provided",
				Paths:   []string{"name"},
			}
		}
		if s.Files == "" {
			return &apis.FieldError{
				Message: "files to stash must be provided",
				Paths:   []string{"files"},
			}
		}
	}

	return nil
}

func validateWorkspace(w string) *apis.FieldError {
	if w == "" {
		return &apis.FieldError{
			Message: "The workspace name must be unspecified or non-empty",
			Paths:   []string{"workspace"},
		}
	}

	return nil
}

// EnvMapToSlice transforms a map of environment variables into a slice that can be used in container configuration
func EnvMapToSlice(envMap map[string]corev1.EnvVar) []corev1.EnvVar {
	env := make([]corev1.EnvVar, 0, len(envMap))

	// Avoid nondeterministic results by sorting the keys and appending vars in that order.
	var envVars []string
	for k := range envMap {
		envVars = append(envVars, k)
	}
	sort.Strings(envVars)

	for _, envVar := range envVars {
		env = append(env, envMap[envVar])
	}

	return env
}

func toContainerEnvVars(origEnv []EnvVar) []corev1.EnvVar {
	env := make([]corev1.EnvVar, 0, len(origEnv))
	for _, e := range origEnv {
		env = append(env, corev1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}

	return env
}

// AddContainerEnvVarsToPipeline allows for adding a slice of container environment variables directly to the
// pipeline, if they're not already defined.
func (j *ParsedPipeline) AddContainerEnvVarsToPipeline(origEnv []corev1.EnvVar) {
	if len(origEnv) > 0 {
		envMap := make(map[string]EnvVar)

		// Add the container env vars first.
		for _, e := range origEnv {
			if e.ValueFrom == nil {
				envMap[e.Name] = EnvVar{
					Name:  e.Name,
					Value: e.Value,
				}
			}
		}

		// Overwrite with the existing pipeline environment, if it exists
		for _, e := range j.GetEnv() {
			envMap[e.Name] = e
		}

		env := make([]EnvVar, 0, len(envMap))

		// Avoid nondeterministic results by sorting the keys and appending vars in that order.
		var envVars []string
		for k := range envMap {
			envVars = append(envVars, k)
		}
		sort.Strings(envVars)

		for _, envVar := range envVars {
			env = append(env, envMap[envVar])
		}

		j.Env = env
	}
}

func scopedEnv(newEnv []corev1.EnvVar, parentEnv []corev1.EnvVar) []corev1.EnvVar {
	if len(parentEnv) == 0 && len(newEnv) == 0 {
		return nil
	}
	envMap := make(map[string]corev1.EnvVar)

	for _, e := range parentEnv {
		envMap[e.Name] = e
	}

	for _, e := range newEnv {
		envMap[e.Name] = e
	}

	return EnvMapToSlice(envMap)
}

func (j *ParsedPipeline) toStepEnvVars() []corev1.EnvVar {
	envMap := make(map[string]corev1.EnvVar)

	for _, e := range j.GetEnv() {
		envMap[e.Name] = corev1.EnvVar{Name: e.Name, Value: e.Value}
	}

	return EnvMapToSlice(envMap)
}

type transformedStage struct {
	Stage Stage
	// Only one of Sequential, Parallel, and Task is non-empty
	Sequential []*transformedStage
	Parallel   []*transformedStage
	Task       *tektonv1alpha1.Task
	// PipelineTask is non-empty only if Task is non-empty, but it is populated
	// after Task is populated so the reverse is not true.
	PipelineTask *tektonv1alpha1.PipelineTask
	// The depth of this stage in the full tree of stages
	Depth int8
	// The parallel or sequntial stage enclosing this stage, or nil if this stage is at top level
	EnclosingStage *transformedStage
	// The stage immediately before this stage at the same depth, or nil if there is no such stage
	PreviousSiblingStage *transformedStage
	// TODO: Add the equivalent reverse relationship
}

func (ts transformedStage) toPipelineStructureStage() v1.PipelineStructureStage {
	s := v1.PipelineStructureStage{
		Name:  ts.Stage.Name,
		Depth: ts.Depth,
	}

	if ts.EnclosingStage != nil {
		s.Parent = &ts.EnclosingStage.Stage.Name
	}

	if ts.PreviousSiblingStage != nil {
		s.Previous = &ts.PreviousSiblingStage.Stage.Name
	}
	// TODO: Add the equivalent reverse relationship

	if ts.PipelineTask != nil {
		s.TaskRef = &ts.PipelineTask.TaskRef.Name
	}

	if len(ts.Parallel) > 0 {
		for _, n := range ts.Parallel {
			s.Parallel = append(s.Parallel, n.Stage.Name)
		}
	}

	if len(ts.Sequential) > 0 {
		for _, n := range ts.Sequential {
			s.Stages = append(s.Stages, n.Stage.Name)
		}
	}

	return s
}

func (ts transformedStage) getAllAsPipelineStructureStages() []v1.PipelineStructureStage {
	var stages []v1.PipelineStructureStage

	stages = append(stages, ts.toPipelineStructureStage())

	if len(ts.Parallel) > 0 {
		for _, n := range ts.Parallel {
			stages = append(stages, n.getAllAsPipelineStructureStages()...)
		}
	}

	if len(ts.Sequential) > 0 {
		for _, n := range ts.Sequential {
			stages = append(stages, n.getAllAsPipelineStructureStages()...)
		}
	}

	return stages
}

func (ts transformedStage) isSequential() bool {
	return len(ts.Sequential) > 0
}

func (ts transformedStage) isParallel() bool {
	return len(ts.Parallel) > 0
}

func (ts transformedStage) getLinearTasks() []*tektonv1alpha1.Task {
	if ts.isSequential() {
		var tasks []*tektonv1alpha1.Task
		for _, seqTs := range ts.Sequential {
			tasks = append(tasks, seqTs.getLinearTasks()...)
		}
		return tasks
	} else if ts.isParallel() {
		var tasks []*tektonv1alpha1.Task
		for _, parTs := range ts.Parallel {
			tasks = append(tasks, parTs.getLinearTasks()...)
		}
		return tasks
	} else {
		return []*tektonv1alpha1.Task{ts.Task}
	}
}

// If the workspace is nil, sets it to the parent's workspace
func (ts *transformedStage) computeWorkspace(parentWorkspace string) {
	if ts.Stage.Options.Workspace == nil {
		ts.Stage.Options.Workspace = &parentWorkspace
	}
}

func stageToTask(s Stage, pipelineIdentifier string, buildIdentifier string, namespace string, wsPath string, parentEnv []corev1.EnvVar, parentAgent Agent, parentWorkspace string, parentContainer *corev1.Container, depth int8, enclosingStage *transformedStage, previousSiblingStage *transformedStage, podTemplates map[string]*corev1.Pod) (*transformedStage, error) {
	if len(s.Post) != 0 {
		return nil, errors.New("post on stages not yet supported")
	}

	stageContainer := &corev1.Container{}

	if !equality.Semantic.DeepEqual(s.Options, StageOptions{}) {
		o := s.Options
		if !equality.Semantic.DeepEqual(o.Timeout, Timeout{}) {
			return nil, errors.New("Timeout on stage not yet supported")
		}
		if o.Retry != 0 {
			return nil, errors.New("Retry on stage not yet supported")
		}
		if !equality.Semantic.DeepEqual(o.Stash, Stash{}) {
			return nil, errors.New("Stash on stage not yet supported")
		}
		if !equality.Semantic.DeepEqual(o.Unstash, Unstash{}) {
			return nil, errors.New("Unstash on stage not yet supported")
		}

		stageContainer = o.ContainerOptions
	}

	if parentContainer != nil {
		merged, err := mergeContainers(parentContainer, stageContainer)
		if err != nil {
			return nil, errors.Wrapf(err, "Error merging stage and parent container overrides: %s", err)
		}
		stageContainer = merged
	}

	env := scopedEnv(toContainerEnvVars(s.GetEnv()), parentEnv)

	agent := s.Agent

	if equality.Semantic.DeepEqual(agent, Agent{}) {
		agent = parentAgent
	}

	stepCounter := 0
	defaultTaskSpec, err := getDefaultTaskSpec(env, stageContainer)
	if err != nil {
		return nil, err
	}

	if len(s.Steps) > 0 {
		t := &tektonv1alpha1.Task{
			TypeMeta: metav1.TypeMeta{
				APIVersion: TektonAPIVersion,
				Kind:       "Task",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      MangleToRfc1035Label(fmt.Sprintf("%s-%s", pipelineIdentifier, s.Name), buildIdentifier),
				Labels:    util.MergeMaps(map[string]string{LabelStageName: s.stageLabelName()}),
			},
		}
		// Only add the default git merge step if this is the first actual step stage - including if the stage is one of
		// N stages within a parallel stage, and that parallel stage is the first stage in the pipeline
		if previousSiblingStage == nil && isNestedFirstStepsStage(enclosingStage) {
			t.Spec = defaultTaskSpec
		}

		t.SetDefaults(context.Background())

		ws := &tektonv1alpha1.TaskResource{
			Name:       "workspace",
			TargetPath: wsPath,
			Type:       tektonv1alpha1.PipelineResourceTypeGit,
		}

		t.Spec.Inputs = &tektonv1alpha1.Inputs{
			Resources: []tektonv1alpha1.TaskResource{*ws},
		}

		t.Spec.Outputs = &tektonv1alpha1.Outputs{
			Resources: []tektonv1alpha1.TaskResource{
				{
					Name: "workspace",
					Type: tektonv1alpha1.PipelineResourceTypeGit,
				},
			},
		}

		// We don't want to dupe volumes for the Task if there are multiple steps
		volumes := make(map[string]corev1.Volume)
		for _, step := range s.Steps {
			actualSteps, stepVolumes, newCounter, err := generateSteps(step, agent.Image, env, stageContainer, podTemplates, stepCounter)
			if err != nil {
				return nil, err
			}

			stepCounter = newCounter

			t.Spec.Steps = append(t.Spec.Steps, actualSteps...)
			for k, v := range stepVolumes {
				volumes[k] = v
			}
		}

		// Avoid nondeterministic results by sorting the keys and appending volumes in that order.
		var volNames []string
		for k := range volumes {
			volNames = append(volNames, k)
		}
		sort.Strings(volNames)

		for _, v := range volNames {
			t.Spec.Volumes = append(t.Spec.Volumes, volumes[v])
		}

		ts := transformedStage{Stage: s, Task: t, Depth: depth, EnclosingStage: enclosingStage, PreviousSiblingStage: previousSiblingStage}
		ts.computeWorkspace(parentWorkspace)
		return &ts, nil
	}
	if len(s.Stages) > 0 {
		var tasks []*transformedStage
		ts := transformedStage{Stage: s, Depth: depth, EnclosingStage: enclosingStage, PreviousSiblingStage: previousSiblingStage}
		ts.computeWorkspace(parentWorkspace)

		for i, nested := range s.Stages {
			var nestedPreviousSibling *transformedStage
			if i > 0 {
				nestedPreviousSibling = tasks[i-1]
			}
			nestedTask, err := stageToTask(nested, pipelineIdentifier, buildIdentifier, namespace, wsPath, env, agent, *ts.Stage.Options.Workspace, stageContainer, depth+1, &ts, nestedPreviousSibling, podTemplates)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, nestedTask)
		}
		ts.Sequential = tasks

		return &ts, nil
	}

	if len(s.Parallel) > 0 {
		var tasks []*transformedStage
		ts := transformedStage{Stage: s, Depth: depth, EnclosingStage: enclosingStage, PreviousSiblingStage: previousSiblingStage}
		ts.computeWorkspace(parentWorkspace)

		for _, nested := range s.Parallel {
			nestedTask, err := stageToTask(nested, pipelineIdentifier, buildIdentifier, namespace, wsPath, env, agent, *ts.Stage.Options.Workspace, stageContainer, depth+1, &ts, nil, podTemplates)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, nestedTask)
		}
		ts.Parallel = tasks

		return &ts, nil
	}
	return nil, errors.New("no steps, sequential stages, or parallel stages")
}

func mergeContainers(parentContainer, childContainer *corev1.Container) (*corev1.Container, error) {
	if parentContainer == nil {
		return childContainer, nil
	} else if childContainer == nil {
		return parentContainer, nil
	}

	// We need JSON bytes to generate a patch to merge the child containers onto the parent container, so marshal the parent.
	parentAsJSON, err := json.Marshal(parentContainer)
	if err != nil {
		return nil, err
	}
	// We need to do a three-way merge to actually combine the parent and child containers, so we need an empty container
	// as the "original"
	emptyAsJSON, err := json.Marshal(&corev1.Container{})
	if err != nil {
		return nil, err
	}
	// Marshal the child to JSON
	childAsJSON, err := json.Marshal(childContainer)
	if err != nil {
		return nil, err
	}

	// Get the patch meta for Container, which is needed for generating and applying the merge patch.
	patchSchema, err := strategicpatch.NewPatchMetaFromStruct(parentContainer)

	if err != nil {
		return nil, err
	}

	// Create a merge patch, with the empty JSON as the original, the child JSON as the modified, and the parent
	// JSON as the current - this lets us do a deep merge of the parent and child containers, with awareness of
	// the "patchMerge" tags.
	patch, err := strategicpatch.CreateThreeWayMergePatch(emptyAsJSON, childAsJSON, parentAsJSON, patchSchema, true)
	if err != nil {
		return nil, err
	}

	// Actually apply the merge patch to the parent JSON.
	mergedAsJSON, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(parentAsJSON, patch, patchSchema)
	if err != nil {
		return nil, err
	}

	// Unmarshal the merged JSON to a Container pointer, and return it.
	merged := &corev1.Container{}
	err = json.Unmarshal(mergedAsJSON, merged)
	if err != nil {
		return nil, err
	}

	return merged, nil
}

func isNestedFirstStepsStage(enclosingStage *transformedStage) bool {
	if enclosingStage != nil {
		if enclosingStage.PreviousSiblingStage != nil {
			return false
		}
		return isNestedFirstStepsStage(enclosingStage.EnclosingStage)
	}
	return true
}

func generateSteps(step Step, inheritedAgent string, env []corev1.EnvVar, parentContainer *corev1.Container, podTemplates map[string]*corev1.Pod, stepCounter int) ([]corev1.Container, map[string]corev1.Volume, int, error) {
	volumes := make(map[string]corev1.Volume)
	var steps []corev1.Container

	stepImage := inheritedAgent
	if step.GetImage() != "" {
		stepImage = step.GetImage()
	}

	workingDir := step.Dir
	if workingDir == "" {
		// TODO: Should be using SourceName from step_create_task, but initial experiments there ended up with some null cases.
		workingDir = "/workspace/source"
	}
	if step.Command != "" {
		c := &corev1.Container{}
		if parentContainer != nil {
			c = parentContainer.DeepCopy()
		}
		if podTemplates != nil && podTemplates[stepImage] != nil {
			podTemplate := podTemplates[stepImage]
			containers := podTemplate.Spec.Containers
			for _, volume := range podTemplate.Spec.Volumes {
				volumes[volume.Name] = volume
			}
			if !equality.Semantic.DeepEqual(c, &corev1.Container{}) {
				merged, err := mergeContainers(&containers[0], c)
				if err != nil {
					return nil, nil, stepCounter, errors.Wrapf(err, "Error merging pod template and parent container: %s", err)
				}
				c = merged
			} else {
				c = &containers[0]
			}
		} else {
			c.Image = stepImage
			c.Command = []string{"/bin/sh", "-c"}
		}
		// Special-casing for commands starting with /kaniko
		// TODO: Should this be more general?
		if strings.HasPrefix(step.Command, "/kaniko") {
			c.Command = []string{step.Command}
			c.Args = step.Arguments
		} else {
			cmdStr := step.Command
			if len(step.Arguments) > 0 {
				cmdStr += " " + strings.Join(step.Arguments, " ")
			}
			c.Args = []string{cmdStr}
		}
		c.WorkingDir = workingDir
		stepCounter++
		if step.Name != "" {
			c.Name = MangleToRfc1035Label(step.Name, "")
		} else {
			c.Name = "step" + strconv.Itoa(1+stepCounter)
		}

		c.Stdin = false
		c.TTY = false
		c.Env = scopedEnv(toContainerEnvVars(step.Env), scopedEnv(env, c.Env))

		steps = append(steps, *c)
	} else if step.Loop != nil {
		for _, v := range step.Loop.Values {
			loopEnv := scopedEnv([]corev1.EnvVar{{Name: step.Loop.Variable, Value: v}}, env)

			for _, s := range step.Loop.Steps {
				loopSteps, loopVolumes, loopCounter, loopErr := generateSteps(s, stepImage, loopEnv, parentContainer, podTemplates, stepCounter)
				if loopErr != nil {
					return nil, nil, loopCounter, loopErr
				}

				// Bump the step counter to what we got from the loop
				stepCounter = loopCounter

				// Add the loop-generated steps
				steps = append(steps, loopSteps...)

				// Add any new volumes that may have shown up
				for k, v := range loopVolumes {
					volumes[k] = v
				}
			}
		}
	} else {
		return nil, nil, stepCounter, errors.New("syntactic sugar steps not yet supported")
	}

	return steps, volumes, stepCounter, nil
}

// PipelineRunName returns the pipeline name given the pipeline and build identifier
func PipelineRunName(pipelineIdentifier string, buildIdentifier string) string {
	return MangleToRfc1035Label(fmt.Sprintf("%s", pipelineIdentifier), buildIdentifier)
}

// GenerateCRDs translates the Pipeline structure into the corresponding Pipeline and Task CRDs
func (j *ParsedPipeline) GenerateCRDs(pipelineIdentifier string, buildIdentifier string, namespace string, podTemplates map[string]*corev1.Pod, taskParams []tektonv1alpha1.TaskParam, sourceDir string) (*tektonv1alpha1.Pipeline, []*tektonv1alpha1.Task, *v1.PipelineStructure, error) {
	if len(j.Post) != 0 {
		return nil, nil, nil, errors.New("Post at top level not yet supported")
	}

	var parentContainer *corev1.Container

	if !equality.Semantic.DeepEqual(j.Options, RootOptions{}) {
		o := j.Options
		if o.Retry != 0 {
			return nil, nil, nil, errors.New("Retry at top level not yet supported")
		}
		parentContainer = o.ContainerOptions
	}

	p := &tektonv1alpha1.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: TektonAPIVersion,
			Kind:       "Pipeline",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      PipelineRunName(pipelineIdentifier, buildIdentifier),
		},
		Spec: tektonv1alpha1.PipelineSpec{
			Resources: []tektonv1alpha1.PipelineDeclaredResource{
				{
					Name: pipelineIdentifier,
					Type: tektonv1alpha1.PipelineResourceTypeGit,
				},
			},
		},
	}

	p.SetDefaults(context.Background())

	structure := &v1.PipelineStructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.Name,
		},
	}

	var previousStage *transformedStage

	var tasks []*tektonv1alpha1.Task

	baseEnv := j.toStepEnvVars()

	for i, s := range j.Stages {
		isLastStage := i == len(j.Stages)-1

		stage, err := stageToTask(s, pipelineIdentifier, buildIdentifier, namespace, sourceDir, baseEnv, j.Agent, "default", parentContainer, 0, nil, previousStage, podTemplates)
		if err != nil {
			return nil, nil, nil, err
		}

		previousStage = stage

		pipelineTasks := createPipelineTasks(stage, p.Spec.Resources[0].Name)

		linearTasks := stage.getLinearTasks()

		for index, lt := range linearTasks {
			if shouldRemoveWorkspaceOutput(stage, lt.Name, index, len(linearTasks), isLastStage) {
				pipelineTasks[index].Resources.Outputs = nil
				lt.Spec.Outputs = nil
			}
			if len(lt.Spec.Inputs.Params) == 0 {
				lt.Spec.Inputs.Params = taskParams
			}
		}

		tasks = append(tasks, linearTasks...)
		p.Spec.Tasks = append(p.Spec.Tasks, pipelineTasks...)
		structure.Stages = append(structure.Stages, stage.getAllAsPipelineStructureStages()...)
	}

	return p, tasks, structure, nil
}

func shouldRemoveWorkspaceOutput(stage *transformedStage, taskName string, index int, tasksLen int, isLastStage bool) bool {
	if stage.isParallel() {
		parallelStages := stage.Parallel
		for _, ps := range parallelStages {
			if ps.Task != nil && ps.Task.Name == taskName {
				return true
			}
			seq := ps.Sequential
			if len(seq) > 0 {
				lastSeq := seq[len(seq)-1]
				if lastSeq.Task.Name == taskName {
					return true
				}
			}

		}
	} else if index == tasksLen-1 && isLastStage {
		return true
	}
	return false
}

func createPipelineTasks(stage *transformedStage, resourceName string) []tektonv1alpha1.PipelineTask {
	if stage.isSequential() {
		var pTasks []tektonv1alpha1.PipelineTask
		for _, nestedStage := range stage.Sequential {
			pTasks = append(pTasks, createPipelineTasks(nestedStage, resourceName)...)
		}
		return pTasks
	} else if stage.isParallel() {
		var pTasks []tektonv1alpha1.PipelineTask
		for _, nestedStage := range stage.Parallel {
			pTasks = append(pTasks, createPipelineTasks(nestedStage, resourceName)...)
		}
		return pTasks
	} else {
		pTask := tektonv1alpha1.PipelineTask{
			Name: stage.Stage.taskName(), // TODO: What should this actually be named?
			TaskRef: tektonv1alpha1.TaskRef{
				Name: stage.Task.Name,
			},
		}

		_, provider := findWorkspaceProvider(stage, stage.getEnclosing(0))
		var previousStageNames []string
		for _, previousStage := range findPreviousNonBlockStages(*stage) {
			previousStageNames = append(previousStageNames, previousStage.PipelineTask.Name)
		}
		pTask.Resources = &tektonv1alpha1.PipelineTaskResources{
			Inputs: []tektonv1alpha1.PipelineTaskInputResource{
				{
					Name:     "workspace",
					Resource: resourceName,
					From:     provider,
				},
			},
			Outputs: []tektonv1alpha1.PipelineTaskOutputResource{
				{
					Name:     "workspace",
					Resource: resourceName,
				},
			},
		}
		pTask.RunAfter = previousStageNames
		stage.PipelineTask = &pTask

		return []tektonv1alpha1.PipelineTask{pTask}
	}
}

// Looks for the most recent Task using the desired workspace that was not in the
// same parallel stage and returns the name of the corresponding Task.
func findWorkspaceProvider(stage, sibling *transformedStage) (bool, []string) {
	if *stage.Stage.Options.Workspace == "empty" {
		return true, nil
	}

	for sibling != nil {
		if sibling.isSequential() {
			found, provider := findWorkspaceProvider(stage, sibling.Sequential[len(sibling.Sequential)-1])
			if found {
				return true, provider
			}
		} else if sibling.isParallel() {
			// We don't want to use a workspace from a parallel stage outside of that stage,
			// but we do need to descend inwards in case stage is in that same stage.
			if stage.getEnclosing(sibling.Depth) == sibling {
				for _, nested := range sibling.Parallel {
					// Pick the parallel branch that has stage
					if stage.getEnclosing(nested.Depth) == nested {
						found, provider := findWorkspaceProvider(stage, nested)
						if found {
							return true, provider
						}
					}
				}
			}
			// TODO: What to do about custom workspaces? Check for erroneous uses specially?
			// Allow them if only one of the parallel tasks uses the same resource?
		} else if sibling.PipelineTask != nil {
			if *sibling.Stage.Options.Workspace == *stage.Stage.Options.Workspace {
				return true, []string{sibling.PipelineTask.Name}
			}
		} else {
			// We are in a sequential stage and sibling has not had its PipelineTask created.
			// Check the task before it so we don't use a workspace of a later task.
		}
		sibling = sibling.PreviousSiblingStage
	}

	return false, nil
}

// Find the end tasks for this stage, traversing down to the end stages of any
// nested sequential or parallel stages as well.
func findEndStages(stage transformedStage) []*transformedStage {
	if stage.isSequential() {
		return findEndStages(*stage.Sequential[len(stage.Sequential)-1])
	} else if stage.isParallel() {
		var endTasks []*transformedStage
		for _, pStage := range stage.Parallel {
			endTasks = append(endTasks, findEndStages(*pStage)...)
		}
		return endTasks
	} else {
		return []*transformedStage{&stage}
	}
}

// Find the tasks that run immediately before this stage, not including
// sequential or parallel wrapper stages.
func findPreviousNonBlockStages(stage transformedStage) []*transformedStage {
	if stage.PreviousSiblingStage != nil {
		return findEndStages(*stage.PreviousSiblingStage)
	} else if stage.EnclosingStage != nil {
		return findPreviousNonBlockStages(*stage.EnclosingStage)
	} else {
		return []*transformedStage{}
	}
}

// Return the stage that encloses this stage at the given depth, or nil if there is no such stage.
// Depth must be >= 0. Returns the stage itself if depth == stage.Depth
func (ts *transformedStage) getEnclosing(depth int8) *transformedStage {
	if ts.Depth == depth {
		return ts
	} else if ts.EnclosingStage == nil {
		return nil
	} else {
		return ts.EnclosingStage.getEnclosing(depth)
	}
}

// Return the first stage that will execute before this stage
// Depth must be >= 0
func (ts transformedStage) getClosestAncestor() *transformedStage {
	if ts.PreviousSiblingStage != nil {
		return ts.PreviousSiblingStage
	} else if ts.EnclosingStage == nil {
		return nil
	} else {
		return ts.EnclosingStage.getClosestAncestor()
	}
}

func findDuplicates(names []string) *apis.FieldError {
	// Count members
	counts := make(map[string]int)
	mangled := make(map[string]string)
	for _, v := range names {
		counts[MangleToRfc1035Label(v, "")]++
		mangled[v] = MangleToRfc1035Label(v, "")
	}

	var duplicateNames []string
	for k, v := range mangled {
		if counts[v] > 1 {
			duplicateNames = append(duplicateNames, "'"+k+"'")
		}
	}

	if len(duplicateNames) > 0 {
		// Avoid nondeterminism in error messages
		sort.Strings(duplicateNames)
		return &apis.FieldError{
			Message: "Stage names must be unique",
			Details: "The following stage names are used more than once: " + strings.Join(duplicateNames, ", "),
		}
	}
	return nil
}

func validateStageNames(j *ParsedPipeline) (err *apis.FieldError) {
	var validate func(stages []Stage, stageNames *[]string)
	validate = func(stages []Stage, stageNames *[]string) {

		for _, stage := range stages {
			*stageNames = append(*stageNames, stage.Name)
			if len(stage.Stages) > 0 {
				validate(stage.Stages, stageNames)
			}
		}

	}
	var names []string

	validate(j.Stages, &names)

	err = findDuplicates(names)

	return
}

// todo JR lets remove this when we switch tekton to using git merge type pipelineresources
func getDefaultTaskSpec(envs []corev1.EnvVar, parentContainer *corev1.Container) (tektonv1alpha1.TaskSpec, error) {
	v := os.Getenv("BUILDER_JX_IMAGE")
	if v == "" {
		v = GitMergeImage
	}

	childContainer := &corev1.Container{
		Name: "git-merge",
		//Image:   "gcr.io/jenkinsxio/builder-jx:0.1.297",
		Image:      v,
		Command:    []string{"jx"},
		Args:       []string{"step", "git", "merge", "--verbose"},
		WorkingDir: "/workspace/source",
		Env:        envs,
	}

	if parentContainer != nil {
		merged, err := mergeContainers(parentContainer, childContainer)
		if err != nil {
			return tektonv1alpha1.TaskSpec{}, err
		}
		childContainer = merged
	}

	return tektonv1alpha1.TaskSpec{
		Steps: []corev1.Container{*childContainer},
	}, nil
}
