package kube

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PipelineActivityKey struct {
	Name     string
	Pipeline string
	Build    string
}

type PromotePullRequestKey struct {
	PipelineActivityKey

	Environment string
}

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

type PromotePullRequestFn func(*v1.PipelineActivity, *v1.PromotePullRequestStep) error

func (k *PromotePullRequestKey) OnPromotePullRequest(activities typev1.PipelineActivityInterface, fn PromotePullRequestFn) error {
	a, p, added, err := k.GetOrCreatePromotePullRequest(activities)
	if err != nil {
		return err
	}
	p1 := *p
	err = fn(a, p)
	if err != nil {
		return err
	}
	p2 := *p

	if added || p1 != p2 {
		_, err = activities.Update(a)
	}
	return err
}

// GetOrCreatePromotePullRequest gets or creates the PromotePullRequest for the key
func (k *PromotePullRequestKey) GetOrCreatePromotePullRequest(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, *v1.PromotePullRequestStep, bool, error) {
	a, err := k.GetOrCreate(activities)
	if err != nil {
		return nil, nil, false, err
	}
	spec := &a.Spec
	for _, step := range spec.Steps {
		if k.matchesPromotePullRequest(&step) {
			return a, step.PromotePullRequest, false, nil
		}
	}
	// lets add a new step
	step := v1.PipelineActivityStep{
		PromotePullRequest: &v1.PromotePullRequestStep{
			Environment: k.Environment,
		},
	}
	spec.Steps = append(spec.Steps, step)
	return a, step.PromotePullRequest, true, nil
}

func (k *PromotePullRequestKey) matchesPromotePullRequest(step *v1.PipelineActivityStep) bool {
	s := step.PromotePullRequest
	return s != nil && s.Environment == k.Environment
}

func (k *PromotePullRequestKey) OnPromote(activities typev1.PipelineActivityInterface, fn PromotePullRequestFn) error {
	a, p, added, err := k.GetOrCreatePromote(activities)
	if err != nil {
		return err
	}
	p1 := *p
	err = fn(a, p)
	if err != nil {
		return err
	}
	p2 := *p

	if added || p1 != p2 {
		_, err = activities.Update(a)
	}
	return err
}

// GetOrCreatePromote gets or creates the Promote for the key
func (k *PromotePullRequestKey) GetOrCreatePromote(activities typev1.PipelineActivityInterface) (*v1.PipelineActivity, *v1.PromotePullRequestStep, bool, error) {
	a, err := k.GetOrCreate(activities)
	if err != nil {
		return nil, nil, false, err
	}
	spec := &a.Spec
	for _, step := range spec.Steps {
		if k.matchesPromote(&step) {
			return a, step.Promote, false, nil
		}
	}
	// lets add a new step
	step := v1.PipelineActivityStep{
		Promote: &v1.PromotePullRequestStep{
			Environment: k.Environment,
		},
	}
	spec.Steps = append(spec.Steps, step)
	return a, step.Promote, true, nil
}

func (k *PromotePullRequestKey) matchesPromote(step *v1.PipelineActivityStep) bool {
	s := step.Promote
	return s != nil && s.Environment == k.Environment
}
