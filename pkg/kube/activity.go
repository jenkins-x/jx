package kube

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PipelineActivityKey struct {
	Name              string
	Pipeline          string
	Build             string
	Version           string
	BuildURL          string
	BuildLogsURL      string
	ReleaseNotesURL   string
	LastCommitSHA     string
	LastCommitMessage string
	LastCommitURL     string
	GitInfo           *gits.GitRepository
}

func (k *PipelineActivityKey) IsValid() bool {
	return len(k.Name) > 0
}

type PromoteStepActivityKey struct {
	PipelineActivityKey

	Environment    string
	ApplicationURL string
}

type PromotePullRequestFn func(*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PromoteActivityStep, *v1.PromotePullRequestStep) error
type PromoteUpdateFn func(*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PromoteActivityStep, *v1.PromoteUpdateStep) error

type PipelineDetails struct {
	GitOwner      string
	GitRepository string
	BranchName    string
	Pipeline      string
	Build         string
}

// PipelineID is an identifier for a Pipeline.
// A pipeline is typically identified by its owner, repository, and branch with the ID field taking the form
// `<owner>/>repository>/<branch>`
type PipelineID struct {
	ID   string
	Name string
}

// NewPipelineIDFromString creates a new PipelineID, given a pre-built string identifier.
// The string identifier is expected to follow the format `<owner>/>repository>/<branch>`, though this isn't actually
// validated/mandated here.
func NewPipelineIDFromString(id string) PipelineID {
	sanitisedName := strings.Replace(strings.ToLower(id), "/", "-", -1)
	sanitisedName = strings.Replace(sanitisedName, "_", "-", -1)
	pID := PipelineID{
		ID: id,
		//TODO: disabling the encoding of the name, as it doesn't work for some upper case values. Upshot is conflicts on org/repo/branch that differ only in case.
		//See https://github.com/jenkins-x/jx/issues/2551
		//Name: util.EncodeKubernetesName(strings.Replace(id, "/", "-", -1)),
		Name: sanitisedName,
	}
	return pID
}

// NewPipelineID creates a new PipelineID for a given owner, repository, and branch.
func NewPipelineID(owner string, repository string, branch string) PipelineID {
	return NewPipelineIDFromString(fmt.Sprintf("%s/%s/%s", owner, repository, branch))
}

// GetActivityName Builds a Kubernetes-friendly (i.e. a-z,-,. only) name for a specific activity, within a pipeline.
func (p *PipelineID) GetActivityName(activity string) string {
	return fmt.Sprintf("%s-%s", p.Name, activity)
}

// CreatePipelineDetails creates a PipelineDetails object populated from the activity
func CreatePipelineDetails(activity *v1.PipelineActivity) *PipelineDetails {
	spec := &activity.Spec
	repoOwner := spec.GitOwner
	repoName := spec.GitRepository
	buildNumber := spec.Build
	branchName := ""
	pipeline := spec.Pipeline
	if pipeline != "" {
		paths := strings.Split(pipeline, "/")
		if len(paths) > 2 {
			if repoOwner == "" {
				repoOwner = paths[0]
			}
			if repoName == "" {
				repoName = paths[1]
			}
			branchName = paths[2]
		}
	}
	if branchName == "" {
		branchName = "master"
	}
	if pipeline == "" && (repoName != "" && repoOwner != "") {
		pipeline = repoOwner + "/" + repoName + "/" + branchName
	}
	return &PipelineDetails{
		GitOwner:      repoOwner,
		GitRepository: repoName,
		BranchName:    branchName,
		Pipeline:      pipeline,
		Build:         buildNumber,
	}
}

// GenerateBuildNumber generates a new build number for the given pipeline
func GenerateBuildNumber(activities typev1.PipelineActivityInterface, pipelines []*v1.PipelineActivity, pn PipelineID) (string, *v1.PipelineActivity, error) {
	buildCounter := 0
	for _, pipeline := range pipelines {
		if strings.EqualFold(pipeline.Spec.Pipeline, pn.ID) {
			b := pipeline.Spec.Build
			if b != "" {
				bi, err := strconv.Atoi(b)
				if err == nil {
					if bi > buildCounter {
						buildCounter = bi
					}
				}
			}
		}
	}
	buildCounter++
	build := strconv.Itoa(buildCounter)
	name := pn.GetActivityName(build)

	k := &PipelineActivityKey{
		Name:     name,
		Pipeline: pn.ID,
		Build:    build,
	}
	a := &v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PipelineActivitySpec{},
	}
	spec := &a.Spec
	updateActivitySpec(k, spec)

	answer, err := activities.Create(a)
	if err != nil {
		return "", nil, err
	}
	return build, answer, nil
}

// GetOrCreate gets or creates the pipeline activity
func (k *PipelineActivityKey) GetOrCreate(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, bool, error) {
	name := ToValidName(k.Name)
	create := false
	defaultActivity := &v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PipelineActivitySpec{},
	}
	if activities == nil {
		log.Warn("Warning: no PipelineActivities client available!")
		return defaultActivity, create, nil
	}
	a, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		create = true
		a = defaultActivity
	}
	oldSpec := a.Spec
	updateActivity(k, a)
	if create {
		answer, err := activities.Create(a)
		return answer, true, err
	} else {
		if !reflect.DeepEqual(&a.Spec, &oldSpec) {
			answer, err := activities.Update(a)
			return answer, false, err
		}
		return a, false, nil
	}
}

func updateActivity(k *PipelineActivityKey, activity *v1.PipelineActivity) {
	if activity.Labels == nil {
		activity.Labels = make(map[string]string, 4)
	}

	updateActivitySpec(k, &activity.Spec)

	activity.Labels["sourcerepository"] = activity.RepositoryName()
	activity.Labels["branch"] = activity.BranchName()
	activity.Labels["owner"] = activity.RepositoryOwner()
}

func updateActivitySpec(k *PipelineActivityKey, spec *v1.PipelineActivitySpec) {
	if k.Pipeline != "" && spec.Pipeline == "" {
		spec.Pipeline = k.Pipeline
	}
	if k.Build != "" && spec.Build == "" {
		spec.Build = k.Build
	}
	if k.BuildURL != "" && spec.BuildURL == "" {
		spec.BuildURL = k.BuildURL
	}
	if k.BuildLogsURL != "" && spec.BuildLogsURL == "" {
		spec.BuildLogsURL = k.BuildLogsURL
	}
	if k.ReleaseNotesURL != "" && spec.ReleaseNotesURL == "" {
		spec.ReleaseNotesURL = k.ReleaseNotesURL
	}
	if k.LastCommitSHA != "" && spec.LastCommitSHA == "" {
		spec.LastCommitSHA = k.LastCommitSHA
	}
	if k.LastCommitMessage != "" && spec.LastCommitMessage == "" {
		spec.LastCommitMessage = k.LastCommitMessage
	}
	if k.LastCommitURL != "" && spec.LastCommitURL == "" {
		spec.LastCommitURL = k.LastCommitURL
	}
	if k.Version != "" && spec.Version == "" {
		spec.Version = k.Version
	}
	gi := k.GitInfo
	if gi != nil {
		if gi.URL != "" && spec.GitURL == "" {
			spec.GitURL = gi.URL
		}
		if gi.Organisation != "" && spec.GitOwner == "" {
			spec.GitOwner = gi.Organisation
		}
		if gi.Name != "" && spec.GitRepository == "" {
			spec.GitRepository = gi.Name
		}
	}
}

// GetOrCreatePreview gets or creates the Preview step for the key
func (k *PromoteStepActivityKey) GetOrCreatePreview(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PreviewActivityStep, bool, error) {
	a, _, err := k.GetOrCreate(activities)
	if err != nil {
		return nil, nil, nil, false, err
	}
	spec := &a.Spec
	for _, step := range spec.Steps {
		if k.matchesPreview(&step) {
			return a, &step, step.Preview, false, nil
		}
	}
	// if there is no initial release Stage lets add one
	if len(spec.Steps) == 0 {
		endTime := time.Now()
		startTime := endTime.Add(-1 * time.Minute)

		spec.Steps = append(spec.Steps, v1.PipelineActivityStep{
			Kind: v1.ActivityStepKindTypeStage,
			Stage: &v1.StageActivityStep{
				CoreActivityStep: v1.CoreActivityStep{
					StartedTimestamp: &metav1.Time{
						Time: startTime,
					},
					CompletedTimestamp: &metav1.Time{
						Time: endTime,
					},
					Status: v1.ActivityStatusTypeSucceeded,
					Name:   "Release",
				},
			},
		})
	}
	// lets add a new step
	preview := &v1.PreviewActivityStep{
		CoreActivityStep: v1.CoreActivityStep{
			StartedTimestamp: &metav1.Time{
				Time: time.Now(),
			},
		},
		Environment: k.Environment,
	}
	step := v1.PipelineActivityStep{
		Kind:    v1.ActivityStepKindTypePreview,
		Preview: preview,
	}
	spec.Steps = append(spec.Steps, step)
	return a, &spec.Steps[len(spec.Steps)-1], preview, true, nil
}

// GetOrCreateStage gets or creates the step for the given name
func GetOrCreateStage(a *v1.PipelineActivity, stageName string) (*v1.PipelineActivityStep, *v1.StageActivityStep, bool) {
	spec := &a.Spec
	for _, step := range spec.Steps {
		stage := step.Stage
		if stage != nil && stage.Name == stageName {
			return &step, stage, false
		}
	}

	stage := &v1.StageActivityStep{
		CoreActivityStep: v1.CoreActivityStep{
			Name: stageName,
		},
	}
	spec.Steps = append(spec.Steps, v1.PipelineActivityStep{
		Kind:  v1.ActivityStepKindTypeStage,
		Stage: stage,
	})
	return &spec.Steps[len(spec.Steps)-1], stage, true
}

// GetOrCreatePromote gets or creates the Promote step for the key
func (k *PromoteStepActivityKey) GetOrCreatePromote(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PromoteActivityStep, bool, error) {
	a, _, err := k.GetOrCreate(activities)
	if err != nil {
		return nil, nil, nil, false, err
	}
	spec := &a.Spec
	for _, step := range spec.Steps {
		if k.matchesPromote(&step) {
			return a, &step, step.Promote, false, nil
		}
	}
	// if there is no initial release Stage lets add one
	if len(spec.Steps) == 0 {
		endTime := time.Now()
		startTime := endTime.Add(-1 * time.Minute)

		spec.Steps = append(spec.Steps, v1.PipelineActivityStep{
			Kind: v1.ActivityStepKindTypeStage,
			Stage: &v1.StageActivityStep{
				CoreActivityStep: v1.CoreActivityStep{
					StartedTimestamp: &metav1.Time{
						Time: startTime,
					},
					CompletedTimestamp: &metav1.Time{
						Time: endTime,
					},
					Status: v1.ActivityStatusTypeSucceeded,
					Name:   "Release",
				},
			},
		})
	}
	// lets add a new step
	promote := &v1.PromoteActivityStep{
		CoreActivityStep: v1.CoreActivityStep{
			StartedTimestamp: &metav1.Time{
				Time: time.Now(),
			},
		},
		Environment: k.Environment,
	}
	step := v1.PipelineActivityStep{
		Kind:    v1.ActivityStepKindTypePromote,
		Promote: promote,
	}
	spec.Steps = append(spec.Steps, step)
	return a, &spec.Steps[len(spec.Steps)-1], promote, true, nil
}

// GetOrCreatePromotePullRequest gets or creates the PromotePullRequest for the key
func (k *PromoteStepActivityKey) GetOrCreatePromotePullRequest(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PromoteActivityStep, *v1.PromotePullRequestStep, bool, error) {
	a, s, p, created, err := k.GetOrCreatePromote(activities)
	if err != nil {
		return nil, nil, nil, nil, created, err
	}
	if p.PullRequest == nil {
		created = true
		p.PullRequest = &v1.PromotePullRequestStep{
			CoreActivityStep: v1.CoreActivityStep{
				StartedTimestamp: &metav1.Time{
					Time: time.Now(),
				},
			},
		}
	}
	return a, s, p, p.PullRequest, created, err
}

// GetOrCreatePromoteUpdate gets or creates the Promote for the key
func (k *PromoteStepActivityKey) GetOrCreatePromoteUpdate(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PromoteActivityStep, *v1.PromoteUpdateStep, bool, error) {
	a, s, p, created, err := k.GetOrCreatePromote(activities)
	if err != nil {
		return nil, nil, nil, nil, created, err
	}

	// lets check the PR is completed
	if p.PullRequest != nil {
		if p.PullRequest.Status == v1.ActivityStatusTypeNone {
			p.PullRequest.Status = v1.ActivityStatusTypeSucceeded
		}
	}
	if p.Update == nil {
		created = true
		p.Update = &v1.PromoteUpdateStep{
			CoreActivityStep: v1.CoreActivityStep{
				StartedTimestamp: &metav1.Time{
					Time: time.Now(),
				},
			},
		}
	}
	return a, s, p, p.Update, created, err
}

func (k *PromoteStepActivityKey) OnPromotePullRequest(activities typev1.PipelineActivityInterface, fn PromotePullRequestFn) error {
	if !k.IsValid() {
		return nil
	}
	if activities == nil {
		log.Warn("Warning: no PipelineActivities client available!")
		return nil
	}
	a, s, ps, p, added, err := k.GetOrCreatePromotePullRequest(activities)
	if err != nil {
		return err
	}
	p1 := *p
	err = fn(a, s, ps, p)
	if err != nil {
		return err
	}
	p2 := *p

	if added || !reflect.DeepEqual(p1, p2) {
		_, err = activities.Update(a)
	}
	return err
}

func (k *PromoteStepActivityKey) OnPromoteUpdate(activities typev1.PipelineActivityInterface, fn PromoteUpdateFn) error {
	if !k.IsValid() {
		return nil
	}
	if activities == nil {
		log.Warn("Warning: no PipelineActivities client available!")
		return nil
	}
	a, s, ps, p, added, err := k.GetOrCreatePromoteUpdate(activities)
	if err != nil {
		return err
	}
	p1 := asYaml(a)
	if k.ApplicationURL != "" {
		ps.ApplicationURL = k.ApplicationURL
	}
	err = fn(a, s, ps, p)
	if err != nil {
		return err
	}
	if k.ApplicationURL != "" {
		ps.ApplicationURL = k.ApplicationURL
	}
	p2 := asYaml(a)

	if added || p1 == "" || p1 != p2 {
		_, err = activities.Update(a)
	}
	return err
}

func asYaml(activity *v1.PipelineActivity) string {
	data, err := yaml.Marshal(activity)
	if err == nil {
		return string(data)
	}
	log.Warnf("Failed to marshal PipelineActivity to YAML %s: %s", activity.Name, err)
	return ""
}

func (k *PromoteStepActivityKey) matchesPreview(step *v1.PipelineActivityStep) bool {
	s := step.Preview
	return s != nil && s.Environment == k.Environment
}

func (k *PromoteStepActivityKey) matchesPromote(step *v1.PipelineActivityStep) bool {
	s := step.Promote
	return s != nil && s.Environment == k.Environment
}
