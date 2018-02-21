package kube

import (
	"fmt"
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/assert"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

type MockPipelineActivityInterface struct {
	Activities map[string]*v1.PipelineActivity
}

func (m *MockPipelineActivityInterface) Create(p *v1.PipelineActivity) (*v1.PipelineActivity, error) {
	m.Activities[p.Name] = p
	return p, nil
}

func (m *MockPipelineActivityInterface) Update(p *v1.PipelineActivity) (*v1.PipelineActivity, error) {
	m.Activities[p.Name] = p
	return p, nil
}

func (m *MockPipelineActivityInterface) Delete(name string, options *meta_v1.DeleteOptions) error {
	delete(m.Activities, name)
	return nil
}

func (m *MockPipelineActivityInterface) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	m.Activities = map[string]*v1.PipelineActivity{}
	return nil
}

func (m *MockPipelineActivityInterface) Get(name string, options meta_v1.GetOptions) (*v1.PipelineActivity, error) {
	a, ok := m.Activities[name]
	if ok {
		return a, nil
	} else {
		return nil, fmt.Errorf("No such PipelineActivity %s", name)
	}
}

func (m *MockPipelineActivityInterface) List(opts meta_v1.ListOptions) (*v1.PipelineActivityList, error) {
	items := []v1.PipelineActivity{}
	for _, p := range m.Activities {
		items = append(items, *p)
	}
	return &v1.PipelineActivityList{
		Items: items,
	}, nil
}

func (m *MockPipelineActivityInterface) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("TODO")
}

func (m *MockPipelineActivityInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PipelineActivity, err error) {
	return nil, fmt.Errorf("TODO")
}

func TestCreateOrUpdateActivities(t *testing.T) {
	activities := &MockPipelineActivityInterface{
		Activities: map[string]*v1.PipelineActivity{},
	}

	const (
		expectedName        = "demo-2"
		expectedPipeline    = "demo"
		expectedBuild       = "2"
		expectedEnvironment = "staging"
	)

	key := PipelineActivityKey{
		Name:     expectedName,
		Pipeline: expectedPipeline,
		Build:    expectedBuild,
	}

	for i := 1; i < 3; i++ {
		a, err := key.GetOrCreate(activities)
		assert.Nil(t, err)
		assert.Equal(t, expectedName, a.Name)
		spec := &a.Spec
		assert.Equal(t, expectedPipeline, spec.Pipeline)
		assert.Equal(t, expectedBuild, spec.Build)
	}

	// lazy add a PromotePullRequest
	promoteKey := PromotePullRequestKey{
		PipelineActivityKey: key,
		Environment:         expectedEnvironment,
	}

	promotePullRequestStarted := func(a *v1.PipelineActivity, p *v1.PromotePullRequestStep) error {
		assert.NotNil(t, a)
		assert.NotNil(t, p)
		if p.StartedTimestamp == nil {
			p.StartedTimestamp = &meta_v1.Time{
				Time: time.Now(),
			}
		}
		return nil
	}

	err := promoteKey.OnPromotePullRequest(activities, promotePullRequestStarted)
	assert.Nil(t, err)

	promoteStarted := func(a *v1.PipelineActivity, p *v1.PromotePullRequestStep) error {
		assert.NotNil(t, a)
		assert.NotNil(t, p)
		if p.StartedTimestamp == nil {
			p.StartedTimestamp = &meta_v1.Time{
				Time: time.Now(),
			}
		}
		if p.CompletedTimestamp == nil {
			p.CompletedTimestamp = &meta_v1.Time{
				Time: time.Now(),
			}
		}
		return nil
	}

	err = promoteKey.OnPromote(activities, promoteStarted)
	assert.Nil(t, err)

	// lets validate that we added a PromotePullRequest step
	a := activities.Activities[expectedName]
	assert.NotNil(t, a, "should have a PipelineActivity for %s", expectedName)
	steps := a.Spec.Steps
	assert.Equal(t, 2, len(steps), "Should have 2 steps!")
	step := a.Spec.Steps[0]
	assert.NotNil(t, step.PromotePullRequest, "step 0 should have a PromotePullRequest")
	assert.NotNil(t, step.PromotePullRequest.StartedTimestamp, "step 0 should have a PromotePullRequest.StartedTimestamp")
	assert.Nil(t, step.PromotePullRequest.CompletedTimestamp, "step 0 should not have a PromotePullRequest.CompletedTimestamp")

	step = a.Spec.Steps[1]
	assert.NotNil(t, step.Promote, "step 1 should have a PromotePullRequest")
	assert.NotNil(t, step.Promote.StartedTimestamp, "step 1 should have a PromotePullRequest.StartedTimestamp")
	assert.NotNil(t, step.Promote.CompletedTimestamp, "step 1 should have a PromotePullRequest.CompletedTimestamp")

	fmt.Printf("Has PromotePullRequest %#v\n", step.PromotePullRequest)
}
