package kube_test

import (
	"fmt"
	jxfake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	k8s_v1 "k8s.io/api/core/v1"
	"strconv"
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	kube_mocks "k8s.io/client-go/kubernetes/fake"
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

func (m *MockPipelineActivityInterface) Delete(name string, options *metav1.DeleteOptions) error {
	delete(m.Activities, name)
	return nil
}

func (m *MockPipelineActivityInterface) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	m.Activities = map[string]*v1.PipelineActivity{}
	return nil
}

func (m *MockPipelineActivityInterface) Get(name string, options metav1.GetOptions) (*v1.PipelineActivity, error) {
	a, ok := m.Activities[name]
	if ok {
		return a, nil
	}
	return nil, fmt.Errorf("No such PipelineActivity %s", name)
}

func (m *MockPipelineActivityInterface) List(opts metav1.ListOptions) (*v1.PipelineActivityList, error) {
	items := []v1.PipelineActivity{}
	for _, p := range m.Activities {
		items = append(items, *p)
	}
	return &v1.PipelineActivityList{
		Items: items,
	}, nil
}

func (m *MockPipelineActivityInterface) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("TODO")
}

func (m *MockPipelineActivityInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PipelineActivity, err error) {
	return nil, fmt.Errorf("TODO")
}

func TestGenerateBuildNumber(t *testing.T) {
	options := &cmd.CommonOptions{Factory: cmd.NewFactory()}
	cmd.ConfigureTestOptions(options, options.Git(), options.Helm())

	jxClient, ns, err := options.JXClientAndDevNamespace()
	assert.NoError(t, err, "Creating JX client")
	if err != nil {
		return
	}

	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	org := "jstrachan"
	repo := "cheese"
	branch := "master"

	results := []string{}
	expected := []string{}
	for i := 1; i < 4; i++ {
		buildNumberText := strconv.Itoa(i)
		pID := kube.NewPipelineID(repo, org, branch)
		pipelines := getPipelines(activities)
		build, activity, err := kube.GenerateBuildNumber(activities, pipelines, pID)
		if assert.NoError(t, err, "GenerateBuildNumber %d", i) {
			if assert.NotNil(t, activity, "No PipelineActivity returned!") {
				results = append(results, build)
				assert.Equal(t, buildNumberText, activity.Spec.Build, "Build number for PipelineActivity %s", activity.Name)
			}
		}
		expected = append(expected, buildNumberText)
	}
	assert.Equal(t, expected, results, "generated build numbers")
}

func getPipelines(activities typev1.PipelineActivityInterface) []*v1.PipelineActivity {
	pipelineList, _ := activities.List(metav1.ListOptions{})
	pipelines := []*v1.PipelineActivity{}
	for _, pipeline := range pipelineList.Items {
		copy := pipeline
		pipelines = append(pipelines, &copy)
	}
	return pipelines
}

func TestCreateOrUpdateActivities(t *testing.T) {
	t.Parallel()

	nsObj := &k8s_v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "jx-testing",
			Namespace: "testing_ns",
		},
	}

	secret := &k8s_v1.Secret{}
	mockKubeClient := kube_mocks.NewSimpleClientset(nsObj, secret)

	ingressConfig := &k8s_v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: kube.ConfigMapIngressConfig,
		},
		Data: map[string]string{"key1": "value1", "domain": "test-domain", "config.yml": ""},
	}

	mockKubeClient.CoreV1().ConfigMaps(nsObj.Namespace).Create(ingressConfig)
	jxClient := jxfake.NewSimpleClientset()

	const (
		expectedName        = "demo-2"
		expectedPipeline    = "demo"
		expectedBuild       = "2"
		expectedEnvironment = "staging"
	)

	key := kube.PipelineActivityKey{
		Name:     expectedName,
		Pipeline: expectedPipeline,
		Build:    expectedBuild,
	}

	for i := 1; i < 3; i++ {
		a, _, err := key.GetOrCreate(jxClient,nsObj.Namespace)
		assert.Nil(t, err)
		assert.Equal(t, expectedName, a.Name)
		spec := &a.Spec
		assert.Equal(t, expectedPipeline, spec.Pipeline)
		assert.Equal(t, expectedBuild, spec.Build)
	}

	// lazy add a PromotePullRequest
	promoteKey := kube.PromoteStepActivityKey{
		PipelineActivityKey: key,
		Environment:         expectedEnvironment,
	}

	promotePullRequestStarted := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
		assert.NotNil(t, a)
		assert.NotNil(t, p)
		if p.StartedTimestamp == nil {
			p.StartedTimestamp = &metav1.Time{
				Time: time.Now(),
			}
		}
		return nil
	}

	err := promoteKey.OnPromotePullRequest(jxClient, nsObj.Namespace, promotePullRequestStarted)
	assert.Nil(t, err)

	promoteStarted := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromoteUpdateStep) error {
		assert.NotNil(t, a)
		assert.NotNil(t, p)
		kube.CompletePromotionUpdate(a, s, ps, p)
		return nil
	}

	err = promoteKey.OnPromoteUpdate(jxClient, nsObj.Namespace, promoteStarted)
	assert.Nil(t, err)

	// lets validate that we added a PromotePullRequest step
	a, err := jxClient.JenkinsV1().PipelineActivities(nsObj.Namespace).Get(expectedName, metav1.GetOptions{})
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

	//tests.Debugf("Has Promote %#v\n", promote)
}

func TestCreatePipelineDetails(t *testing.T) {
	expectedGitOwner := "jstrachan"
	expectedGitRepo := "myapp"
	expectedBranch := "master"
	expectedPipeline := expectedGitOwner + "/" + expectedGitRepo + "/" + expectedBranch
	expectedBuild := "3"

	pipelines := []*v1.PipelineActivity{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "a1",
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline: expectedPipeline,
				Build:    expectedBuild,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "a2",
			},
			Spec: v1.PipelineActivitySpec{
				GitOwner:      expectedGitOwner,
				GitRepository: expectedGitRepo,
				Build:         expectedBuild,
			},
		},
	}
	for _, pipeline := range pipelines {
		d1 := kube.CreatePipelineDetails(pipeline)
		name := pipeline.Name
		if assert.NotNil(t, d1, "%s did not create a PipelineDetails", name) {
			assert.Equal(t, expectedGitOwner, d1.GitOwner, "%s GitOwner", name)
			assert.Equal(t, expectedGitRepo, d1.GitRepository, "%s GitRepository", name)
			assert.Equal(t, expectedBranch, d1.BranchName, "%s BranchName", name)
			assert.Equal(t, expectedPipeline, d1.Pipeline, "%s Pipeline", name)
			assert.Equal(t, expectedBuild, d1.Build, "%s Build", name)
		}
	}
}

func TestPipelineID(t *testing.T) {
	t.Parallel()

	// A simple ID.
	pID := kube.NewPipelineID("o1", "r1", "b1")
	validatePipelineID(t, pID, "o1/r1/b1", "o1-r1-b1")

	// Upper case allowed in our ID, but not in the K8S 'name'.
	pID = kube.NewPipelineID("OwNeR1", "rEpO1", "BrAnCh1")
	validatePipelineID(t, pID, "OwNeR1/rEpO1/BrAnCh1", "owner1-repo1-branch1")

	//Punctuation other than '-' and '.' not allowed in K8S 'name'. Note that this isn't currently handled by the
	//system - the illegal characters are not yet encoded & will be rejected by K8S.
	pID = kube.NewPipelineID("O/N!R@1", "therepo", "thebranch")
	validatePipelineID(t, pID, "O/N!R@1/therepo/thebranch", "o-n!r@1-therepo-thebranch")
}

func validatePipelineID(t *testing.T, pID kube.PipelineID, expectedID string, expectedName string) {
	assert.Equal(t, expectedID, pID.ID)
	assert.Equal(t, expectedName, pID.Name)
}
