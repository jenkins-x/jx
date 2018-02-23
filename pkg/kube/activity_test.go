package kube

import (
	"fmt"
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/assert"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
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
	promoteKey := PromoteStepActivityKey{
		PipelineActivityKey: key,
		Environment:         expectedEnvironment,
	}

	promotePullRequestStarted := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
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

	promoteStarted := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromoteUpdateStep) error {
		assert.NotNil(t, a)
		assert.NotNil(t, p)
		CompletePromotionUpdate(a, s, ps, p)
		return nil
	}

	err = promoteKey.OnPromoteUpdate(activities, promoteStarted)
	assert.Nil(t, err)

	// lets validate that we added a PromotePullRequest step
	a := activities.Activities[expectedName]
	assert.NotNil(t, a, "should have a PipelineActivity for %s", expectedName)
	steps := a.Spec.Steps
	assert.Equal(t, 2, len(steps), "Should have 2 steps!")
	step := a.Spec.Steps[0]
	stage := step.Stage
	assert.NotNil(t, stage, "step 0 should have a Stage")
	assert.Equal(t, v1.ActivityStepKindTypeStage, step.Kind, "step - kind")
	assert.Equal(t, v1.ActivityStatusTypeSucceeded, stage.Status, "step 0 Stage status")
	assert.NotNil(t, stage.StartedTimestamp, "stage should have a StartedTimestamp")
	assert.NotNil(t, stage.CompletedTimestamp, "stage should have a CompletedTimestamp")

	step = a.Spec.Steps[1]
	promote := step.Promote
	assert.NotNil(t, promote, "step 1 should have a Promote")
	assert.Equal(t, v1.ActivityStepKindTypePromote, step.Kind, "step 1 kind")

	pullRequestStep := promote.PullRequest
	assert.NotNil(t, pullRequestStep, "Promote should have a PullRequest")
	assert.NotNil(t, pullRequestStep.StartedTimestamp, "Promote should have a PullRequest.StartedTimestamp")
	assert.NotNil(t, pullRequestStep.CompletedTimestamp, "Promote should not have a PullRequest.CompletedTimestamp")

	updateStep := promote.Update
	assert.NotNil(t, updateStep, "Promote should have an Update")
	assert.NotNil(t, updateStep.StartedTimestamp, "Promote should have a Update.StartedTimestamp")
	assert.NotNil(t, updateStep.CompletedTimestamp, "Promote should have a Update.CompletedTimestamp")

	assert.NotNil(t, promote.StartedTimestamp, "promote should have a StartedTimestamp")
	assert.NotNil(t, promote.CompletedTimestamp, "promote should have a CompletedTimestamp")

	assert.Equal(t, v1.ActivityStatusTypeSucceeded, pullRequestStep.Status, "pullRequestStep status")
	assert.Equal(t, v1.ActivityStatusTypeSucceeded, updateStep.Status, "updateStep status")
	assert.Equal(t, v1.ActivityStatusTypeSucceeded, promote.Status, "promote status")

	//fmt.Printf("Has Promote %#v\n", promote)
}
