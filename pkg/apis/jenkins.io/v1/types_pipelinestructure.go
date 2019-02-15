package v1

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/knative/pkg/apis"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// PipelineStructure is the internal representation of the Pipeline, used to validate and create CRDs
type PipelineStructure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	// +optional
	Agent *PipelineStructureAgent `json:"agent,omitempty" protobuf:"bytes,2,opt,name=agent"`
	// +optional
	Environment []PipelineStructureEnvVar `json:"environment,omitempty" protobuf:"bytes,3,opt,name=environment"`
	// +optional
	Options *PipelineStructureRootOptions `json:"options,omitempty" protobuf:"bytes,4,opt,name=options"`
	Stages  []*PipelineStructureStage     `json:"stages" protobuf:"bytes,5,opt,name=stages"`
	// +optional
	Post []PipelineStructurePost `json:"post,omitempty" protobuf:"bytes,6,opt,name=post"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineStructureList is a list of PipelineStructure resources
type PipelineStructureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PipelineStructure `json:"items"`
}

// PipelineStructureAgent defines where the pipeline, stage, or step should run.
type PipelineStructureAgent struct {
	// One of label or image is required.
	// +optional
	Label *string `json:"label,omitempty" protobuf:"bytes,1,opt,name=label"`
	// +optional
	Image *string `json:"image,omitempty" protobuf:"bytes,1,opt,name=image"`
	// Perhaps we'll eventually want to add something here for specifying a volume to create? Would play into stash.
}

// PipelineStructureEnvVar is a key/value pair defining an environment variable
type PipelineStructureEnvVar struct {
	Name  string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
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

// PipelineStructureTimeout defines how long a stage or pipeline can run before timing out.
type PipelineStructureTimeout struct {
	Time int64 `json:"time" protobuf:"bytes,1,opt,name=time"`
	// Has some sane default - probably seconds
	// +optional
	Unit *TimeoutUnit `json:"unit,omitempty" protobuf:"bytes,2,opt,name=unit"`
}

func (t PipelineStructureTimeout) toDuration() (*metav1.Duration, error) {
	durationStr := ""
	// TODO: Populate a default timeout unit, most likely seconds.
	if t.Unit != nil {
		durationStr = fmt.Sprintf("%d%c", t.Time, (*t.Unit)[0])
	} else {
		durationStr = fmt.Sprintf("%ds", t.Time)
	}

	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, err
	}
	return &metav1.Duration{Duration: d}, nil
}

// PipelineStructureRootOptions contains options that can be configured on either a pipeline or a stage
type PipelineStructureRootOptions struct {
	// +optional
	Timeout *PipelineStructureTimeout `json:"timeout,omitempty" protobuf:"bytes,1,opt,name=timeout"`
	// TODO: Not yet implemented in build-pipeline
	// +optional
	Retry int8 `json:"retry,omitempty" protobuf:"bytes,2,opt,name=retry"`
}

// PipelineStructureStash defines files to be saved for use in a later stage, marked with a name
type PipelineStructureStash struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Eventually make this optional so that you can do volumes instead
	Files string `json:"files" protobuf:"bytes,2,opt,name=files"`
}

// PipelineStructureUnstash defines a previously-defined stash to be copied into this stage's workspace
type PipelineStructureUnstash struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// +optional
	Dir *string `json:"dir,omitempty" protobuf:"bytes,2,opt,name=dir"`
}

// PipelineStructureStageOptions contains both options that can be configured on either a pipeline or a stage, via
// PipelineStructureRootOptions, or stage-specific options.
type PipelineStructureStageOptions struct {
	// +optional
	Timeout *PipelineStructureTimeout `json:"timeout,omitempty" protobuf:"bytes,1,opt,name=timeout"`
	// TODO: Not yet implemented in build-pipeline
	// +optional
	Retry int8 `json:"retry,omitempty" protobuf:"bytes,2,opt,name=retry"`

	// TODO: Not yet implemented in build-pipeline
	// +optional
	Stash *PipelineStructureStash `json:"stash,omitempty" protobuf:"bytes,3,opt,name=stash"`
	// +optional
	Unstash *PipelineStructureUnstash `json:"unstash,omitempty" protobuf:"bytes,4,opt,name=unstash"`
	// +optional
	Workspace *string `json:"workspace,omitempty" protobuf:"bytes,5,opt,name=workspace"`
}

// PipelineStructureStep defines a single step, from the author's perspective, to be executed within a stage.
type PipelineStructureStep struct {
	// One of command, step, or loop is required.
	// +optional
	Command *string `json:"command,omitempty" protobuf:"bytes,1,opt,name=command"`
	// args is optional, but only allowed with command
	// +optional
	Args []string `json:"args,omitempty" protobuf:"bytes,2,opt,name=args"`
	// dir is optional, but only allowed with command. Refers to subdirectory of workspace
	// +optional
	Dir *string `json:"dir,omitempty" protobuf:"bytes,3,opt,name=dir"`

	// +optional
	Step *string `json:"step,omitempty" protobuf:"bytes,4,opt,name=step"`
	// options is optional, but only allowed with step
	// Also, we'll need to do some magic to do type verification during translation - i.e., this step wants a number
	// for this option, so translate the string value for that option to a number.
	// +optional
	Options map[string]string `json:"options,omitempty" protobuf:"bytes,5,opt,name=options"`

	// +optional
	Loop *PipelineStructureLoop `json:"loop,omitempty" protobuf:"bytes,6,opt,name=loop"`

	// agent can be overridden on a step
	// +optional
	Agent *PipelineStructureAgent `json:"agent,omitempty" protobuf:"bytes,7,opt,name=agent"`
}

// PipelineStructureLoop is a special step that defines a variable, a list of possible values for that variable, and a set of steps to
// repeat for each value for the variable, with the variable set with that value in the environment for the execution of
// those steps.
type PipelineStructureLoop struct {
	// The variable name.
	Variable string `json:"variable" protobuf:"bytes,1,opt,name=variable"`
	// The list of values to iterate over
	Values []string `json:"values" protobuf:"bytes,2,opt,name=values"`
	// The steps to run
	Steps []PipelineStructureStep `json:"steps" protobuf:"bytes,3,opt,name=steps"`
}

// PipelineStructureStage is a unit of work in a pipeline, corresponding either to a Task or a set of Tasks to be run sequentially or in
// parallel with common configuration.
type PipelineStructureStage struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// +optional
	Agent *PipelineStructureAgent `json:"agent,omitempty" protobuf:"bytes,2,opt,name=agent"`
	// +optional
	Options *PipelineStructureStageOptions `json:"options,omitempty" protobuf:"bytes,3,opt,name=options"`
	// +optional
	Environment []PipelineStructureEnvVar `json:"environment,omitempty" protobuf:"bytes,4,opt,name=environment"`
	// +optional
	Steps []PipelineStructureStep `json:"steps,omitempty" protobuf:"bytes,5,opt,name=steps"`
	// +optional
	Stages []*PipelineStructureStage `json:"stages,omitempty" protobuf:"bytes,6,opt,name=stages"`
	// +optional
	Parallel []*PipelineStructureStage `json:"parallel,omitempty" protobuf:"bytes,7,opt,name=parallel"`
	// +optional
	Post []PipelineStructurePost `json:"post,omitempty" protobuf:"bytes,8,opt,name=post"`
	// +optional
	Parent *string `json:"parent,omitempty" protobuf:"bytes,9,opt,name=parent"`
	// +optional
	Depth int8 `json:"depth,omitempty" protobuf:"bytes,10,opt,name=depth"`
}

// PostCondition is used to specify under what condition a post action should be executed.
type PostCondition string

// Probably extensible down the road
const (
	PostConditionSuccess PostCondition = "success"
	PostConditionFailure PostCondition = "failure"
	PostConditionAlways  PostCondition = "always"
)

var allPostConditions = []PostCondition{PostConditionAlways, PostConditionSuccess, PostConditionFailure}

// PipelineStructurePost contains a PostCondition and one more actions to be executed after a pipeline or stage if the condition is met.
type PipelineStructurePost struct {
	// TODO: Conditional execution of something after a Task or Pipeline completes is not yet implemented
	Condition PostCondition                 `json:"condition" protobuf:"bytes,1,opt,name=condition"`
	Actions   []PipelineStructurePostAction `json:"actions" protobuf:"bytes,2,opt,name=actions"`
}

// PipelineStructurePostAction contains the name of a built-in post action and options to pass to that action.
type PipelineStructurePostAction struct {
	// TODO: Notifications are not yet supported in Build Pipeline per se.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Also, we'll need to do some magic to do type verification during translation - i.e., this action wants a number
	// for this option, so translate the string value for that option to a number.
	// +optional
	Options map[string]string `json:"options,omitempty" protobuf:"bytes,2,opt,name=options"`
}

var _ apis.Validatable = (*PipelineStructure)(nil)

// TaskName translates the stage name
func (s *PipelineStructureStage) TaskName() string {
	return strings.ToLower(strings.NewReplacer(" ", "-").Replace(s.Name))
}

// MangleToRfc1035Label - Task/PipelineStructureStep names need to be RFC 1035/1123 compliant DNS labels, so we mangle
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

// Validate checks the parsed PipelineStructure to find any errors in it.
// TODO: Improve validation to actually return all the errors via the nested errors?
// TODO: Add validation for the not-yet-supported-for-CRD-generation sections
func (j *PipelineStructure) Validate() *apis.FieldError {
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

func validateAgent(a *PipelineStructureAgent) *apis.FieldError {
	// TODO: This is the same whether you specify an agent without label or image, or if you don't specify an agent
	// at all, which is nonoptimal.
	if a != nil {
		if a.Image != nil && a.Label != nil {
			return apis.ErrMultipleOneOf("label", "image")
		}

		if a.Image == nil && a.Label == nil {
			return apis.ErrMissingOneOf("label", "image")
		}
	}

	return nil
}

var containsASCIILetter = regexp.MustCompile(`[a-zA-Z]`).MatchString

func validateStage(s *PipelineStructureStage, parentAgent *PipelineStructureAgent) *apis.FieldError {
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
	if stageAgent == nil {
		stageAgent = parentAgent
	}

	if stageAgent == nil {
		return &apis.FieldError{
			Message: "No agent specified for stage or for its parent(s)",
			Paths:   []string{"agent"},
		}
	}

	if len(s.Steps) > 0 {
		if len(s.Stages) > 0 || len(s.Parallel) > 0 {
			return apis.ErrMultipleOneOf("steps", "stages", "parallel")
		}
		for i, step := range s.Steps {
			if err := validateStep(step).ViaFieldIndex("steps", i); err != nil {
				return err
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

func validateStep(s PipelineStructureStep) *apis.FieldError {
	if s.Command == nil && s.Step == nil && s.Loop == nil {
		return apis.ErrMissingOneOf("command", "step", "loop")
	}

	if moreThanOneAreTrue(s.Command != nil, s.Step != nil, s.Loop != nil) {
		return apis.ErrMultipleOneOf("command", "step", "loop")
	}

	if (s.Command != nil || s.Loop != nil) && len(s.Options) != 0 {
		return &apis.FieldError{
			Message: "Cannot set options for a command or a loop",
			Paths:   []string{"options"},
		}
	}

	if (s.Step != nil || s.Loop != nil) && len(s.Args) != 0 {
		return &apis.FieldError{
			Message: "Cannot set command-line arguments for a step or a loop",
			Paths:   []string{"args"},
		}
	}

	if err := validateLoop(s.Loop); err != nil {
		return err.ViaField("loop")
	}

	return validateAgent(s.Agent).ViaField("agent")
}

func validateLoop(l *PipelineStructureLoop) *apis.FieldError {
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

func validateStages(stages []*PipelineStructureStage, parentAgent *PipelineStructureAgent) *apis.FieldError {
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

func validateRootOptions(o *PipelineStructureRootOptions) *apis.FieldError {
	if o != nil {
		if o.Timeout != nil {
			if err := validateTimeout(o.Timeout); err != nil {
				return err.ViaField("timeout")
			}
		}

		if o.Retry < 0 {
			return &apis.FieldError{
				Message: "Retry count cannot be negative",
				Paths:   []string{"retry"},
			}
		}
	}

	return nil
}

func validateStageOptions(o *PipelineStructureStageOptions) *apis.FieldError {
	if o != nil {
		if o.Stash != nil {
			if err := validateStash(o.Stash); err != nil {
				return err.ViaField("stash")
			}
		}

		if o.Unstash != nil {
			if err := validateUnstash(o.Unstash); err != nil {
				return err.ViaField("unstash")
			}
		}

		if o.Workspace != nil {
			if err := validateWorkspace(*o.Workspace); err != nil {
				return err
			}
		}

		if o.Timeout != nil {
			if err := validateTimeout(o.Timeout); err != nil {
				return err.ViaField("timeout")
			}
		}

		if o.Retry < 0 {
			return &apis.FieldError{
				Message: "Retry count cannot be negative",
				Paths:   []string{"retry"},
			}
		}
	}
	return nil
}

func validateTimeout(t *PipelineStructureTimeout) *apis.FieldError {
	if t != nil {
		if t.Unit != nil {
			isAllowed := false
			for _, allowed := range allTimeoutUnits {
				if *t.Unit == allowed {
					isAllowed = true
				}
			}

			if !isAllowed {
				return &apis.FieldError{
					Message: fmt.Sprintf("%s is not a valid time unit. Valid time units are %s", string(*t.Unit),
						strings.Join(allTimeoutUnitsAsStrings(), ", ")),
					Paths: []string{"unit"},
				}
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

func validateUnstash(u *PipelineStructureUnstash) *apis.FieldError {
	if u != nil {
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

func validateStash(s *PipelineStructureStash) *apis.FieldError {
	if s != nil {
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

// ToContainerEnvVars translates the PipelineStructure's environment to env vars suitable for step/container usage.
func (j *PipelineStructure) ToContainerEnvVars() []corev1.EnvVar {
	env := make([]corev1.EnvVar, 0, len(j.Environment))

	for _, e := range j.Environment {
		env = append(env, corev1.EnvVar{Name: e.Name, Value: e.Value})
	}

	return env
}

func findDuplicates(names []string) *apis.FieldError {
	// Count members
	counts := make(map[string]int)
	for _, v := range names {
		counts[v]++
	}

	var duplicateNames []string
	for k, v := range counts {
		if v > 1 {
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

func validateStageNames(j *PipelineStructure) (err *apis.FieldError) {
	var validate func(stages []*PipelineStructureStage, stageNames *[]string)
	validate = func(stages []*PipelineStructureStage, stageNames *[]string) {

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
