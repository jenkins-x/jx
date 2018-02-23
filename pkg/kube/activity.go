package kube

import (
	"reflect"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PipelineActivityKey struct {
	Name     string
	Pipeline string
	Build    string
}

func (k *PipelineActivityKey) IsValid() bool {
	return len(k.Name) > 0
}

type PromoteStepActivityKey struct {
	PipelineActivityKey

	Environment string
}

type PromotePullRequestFn func(*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PromoteActivityStep, *v1.PromotePullRequestStep) error
type PromoteUpdateFn func(*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PromoteActivityStep, *v1.PromoteUpdateStep) error

// GetOrCreate gets or creates the pipeline activity
func (k *PipelineActivityKey) GetOrCreate(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, error) {
	name := k.Name
	a, err := activities.Get(name, metav1.GetOptions{})
	if err == nil {
		return a, nil
	}
	a = &v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PipelineActivitySpec{
			Pipeline: k.Pipeline,
			Build:    k.Build,
		},
	}
	return activities.Create(a)
}

// GetOrCreatePromote gets or creates the Promote step for the key
func (k *PromoteStepActivityKey) GetOrCreatePromote(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, *v1.PipelineActivityStep, *v1.PromoteActivityStep, bool, error) {
	a, err := k.GetOrCreate(activities)
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
	a, s, ps, p, added, err := k.GetOrCreatePromoteUpdate(activities)
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

func (k *PromoteStepActivityKey) matchesPromote(step *v1.PipelineActivityStep) bool {
	s := step.Promote
	return s != nil && s.Environment == k.Environment
}
