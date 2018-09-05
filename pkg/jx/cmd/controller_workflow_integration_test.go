// +build integration

package cmd_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func TestSequentialWorkflow(t *testing.T) {
	testOrgName := "jstrachan"
	testRepoName := "myrepo"
	stagingRepoName := "environment-staging"
	prodRepoName := "environment-production"

	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	stagingRepo := gits.NewFakeRepository(testOrgName, stagingRepoName)
	prodRepo := gits.NewFakeRepository(testOrgName, prodRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)

	o := &cmd.ControllerWorkflowOptions{
		NoWatch:          true,
		FakePullRequests: NewCreateEnvPullRequestFn(fakeGitProvider),
		FakeGitProvider:  fakeGitProvider,
	}

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://github.com/"+testOrgName+"/"+stagingRepoName+".git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://github.com/"+testOrgName+"/"+prodRepoName+".git")
	staging.Spec.Order = 100
	production.Spec.Order = 200

	myFlowName := "myflow"

	step1 := workflow.CreateWorkflowPromoteStep("staging")
	step2 := workflow.CreateWorkflowPromoteStep("production", step1)

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			staging,
			production,
			kube.NewPreviewEnvironment("jx-jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment("jx-jstrachan-another-pr-3"),
			workflow.CreateWorkflow("jx", myFlowName,
				step1,
				step2,
			),
		},
		gits.NewGitCLI(),
		helm.NewHelmCLI("helm", helm.V2, ""),
	)
	o.GitClient = &gits.GitFake{}

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)
	if err == nil {
		workflow, err := workflow.GetWorkflow("", jxClient, ns)
		assert.NoError(t, err)
		if err == nil {
			assert.Equal(t, "default", workflow.Name, "name")
			spec := workflow.Spec
			assert.Equal(t, 2, len(spec.Steps), "number of steps")
			if len(spec.Steps) > 0 {
				assertPromoteStep(t, &spec.Steps[0], "staging")
			}
			if len(spec.Steps) > 1 {
				assertPromoteStep(t, &spec.Steps[1], "production")
			}
		}
	}

	a, err := createTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", myFlowName)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	err = o.Run()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	assertHasPullRequestForEnv(t, activities, a.Name, "staging")
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets make sure we don't create a PR for production as we have not completed the staging PR yet
	err = o.Run()
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// still no PR merged so cannot create a PR for production
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// test no PR on production until staging completed
	if !assertSetPullRequestMerged(t, fakeGitProvider, stagingRepo, 1) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	if !assertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 1) {
		return
	}

	// now lets poll again due to change to the activity to detect the staging is complete
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPullRequestForEnv(t, activities, a.Name, "production")
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	if !assertSetPullRequestMerged(t, fakeGitProvider, prodRepo, 1) {
		return
	}
	if !assertSetPullRequestComplete(t, fakeGitProvider, prodRepo, 1) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeSucceeded)

	assertAllPromoteStepsSuccessful(t, activities, a.Name)
}

func TestWorkflowManualPromote(t *testing.T) {
	testOrgName := "jstrachan"
	testRepoName := "manual"
	stagingRepoName := "environment-staging"
	prodRepoName := "environment-production"

	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	stagingRepo := gits.NewFakeRepository(testOrgName, stagingRepoName)
	prodRepo := gits.NewFakeRepository(testOrgName, prodRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)

	o := &cmd.ControllerWorkflowOptions{
		NoWatch:          true,
		FakePullRequests: NewCreateEnvPullRequestFn(fakeGitProvider),
		FakeGitProvider:  fakeGitProvider,
	}

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://github.com/"+testOrgName+"/"+stagingRepoName+".git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://github.com/"+testOrgName+"/"+prodRepoName+".git")
	production.Spec.PromotionStrategy = v1.PromotionStrategyTypeManual

	workflowName := "default"

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			staging,
			production,
			kube.NewPreviewEnvironment("jx-jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment("jx-jstrachan-another-pr-3"),
		},
		gits.NewGitCLI(),
		helm.NewHelmCLI("helm", helm.V2, ""),
	)
	o.GitClient = &gits.GitFake{}

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)

	a, err := createTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", workflowName)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	err = o.Run()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	assertHasPullRequestForEnv(t, activities, a.Name, "staging")
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets make sure we don't create a PR for production as its manual
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	if !assertSetPullRequestMerged(t, fakeGitProvider, stagingRepo, 1) {
		return
	}
	if !assertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 1) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")
	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertAllPromoteStepsSuccessful(t, activities, a.Name)

	// now lets do a manual promotion
	version := a.Spec.Version
	po := &cmd.PromoteOptions{
		Application:       testRepoName,
		Environment:       "production",
		Pipeline:          a.Spec.Pipeline,
		Build:             a.Spec.Build,
		Version:           version,
		NoPoll:            true,
		IgnoreLocalFiles:  true,
		HelmRepositoryURL: helm.DefaultHelmRepositoryURL,
		LocalHelmRepoName: kube.LocalHelmRepoName,
		FakePullRequests:  o.FakePullRequests,
	}
	po.CommonOptions = o.CommonOptions
	po.BatchMode = true
	log.Infof("Promoting to production version %s for app %s\n", version, testRepoName)
	err = po.Run()
	assert.NoError(t, err)
	if err != nil {
		return
	}

	assertHasPullRequestForEnv(t, activities, a.Name, "production")
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	if !assertSetPullRequestMerged(t, fakeGitProvider, prodRepo, 1) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	if !assertSetPullRequestComplete(t, fakeGitProvider, prodRepo, 1) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeSucceeded)
	assertAllPromoteStepsSuccessful(t, activities, a.Name)
}

func dumpActivity(t *testing.T, activities typev1.PipelineActivityInterface, name string) *v1.PipelineActivity {
	activity, err := activities.Get(name, metav1.GetOptions{})
	assert.NoError(t, err)
	if err != nil {
		return nil
	}
	assert.NotNil(t, activity, "No PipelineActivity found for name %s", name)
	if activity != nil {
		dumpFailedActivity(activity)
	}
	return activity
}

// TestParallelWorkflow lets test promoting to A + B then when A + B is complete then C
func TestParallelWorkflow(t *testing.T) {
	testOrgName := "jstrachan"
	testRepoName := "parallelrepo"

	envNameA := "a"
	envNameB := "b"
	envNameC := "c"

	envRepoNameA := "environment-" + envNameA
	envRepoNameB := "environment-" + envNameB
	envRepoNameC := "environment-" + envNameC

	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	repoA := gits.NewFakeRepository(testOrgName, envRepoNameA)
	repoB := gits.NewFakeRepository(testOrgName, envRepoNameB)
	repoC := gits.NewFakeRepository(testOrgName, envRepoNameC)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, repoA, repoB, repoC)

	o := &cmd.ControllerWorkflowOptions{
		NoWatch:          true,
		FakePullRequests: NewCreateEnvPullRequestFn(fakeGitProvider),
		FakeGitProvider:  fakeGitProvider,
	}

	envA := kube.NewPermanentEnvironmentWithGit(envNameA, "https://github.com/"+testOrgName+"/"+envRepoNameA+".git")
	envB := kube.NewPermanentEnvironmentWithGit(envNameB, "https://github.com/"+testOrgName+"/"+envRepoNameB+".git")
	envC := kube.NewPermanentEnvironmentWithGit(envNameC, "https://github.com/"+testOrgName+"/"+envRepoNameC+".git")

	myFlowName := "myflow"

	step1 := workflow.CreateWorkflowPromoteStep(envNameA)
	step2 := workflow.CreateWorkflowPromoteStep(envNameB)
	step3 := workflow.CreateWorkflowPromoteStep(envNameC, step1, step2)

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			envA,
			envB,
			envC,
			kube.NewPreviewEnvironment("jx-jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment("jx-jstrachan-another-pr-3"),
			workflow.CreateWorkflow("jx", myFlowName,
				step1,
				step2,
				step3,
			),
		},
		gits.NewGitCLI(),
		helm.NewHelmCLI("helm", helm.V2, ""),
	)
	o.GitClient = &gits.GitFake{}

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)
	if err == nil {
		workflow, err := workflow.GetWorkflow("", jxClient, ns)
		assert.NoError(t, err)
		if err == nil {
			assert.Equal(t, "default", workflow.Name, "name")
			spec := workflow.Spec
			assert.Equal(t, 3, len(spec.Steps), "number of steps")
			if len(spec.Steps) > 0 {
				assertPromoteStep(t, &spec.Steps[0], envNameA)
			}
			if len(spec.Steps) > 1 {
				assertPromoteStep(t, &spec.Steps[1], envNameB)
			}
			if len(spec.Steps) > 2 {
				assertPromoteStep(t, &spec.Steps[2], envNameC)
			}
		}
	}

	a, err := createTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", myFlowName)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	err = o.Run()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	assertHasPullRequestForEnv(t, activities, a.Name, envNameA)
	assertHasPullRequestForEnv(t, activities, a.Name, envNameB)
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets make sure we don't create a PR for production as we have not completed the staging PR yet
	err = o.Run()
	assertHasNoPullRequestForEnv(t, activities, a.Name, envNameC)

	// still no PR merged so cannot create a PR for C until A and B complete
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, envNameC)

	// test no PR on production until staging completed
	if !assertSetPullRequestMerged(t, fakeGitProvider, repoA, 1) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, envNameC)

	if !assertSetPullRequestComplete(t, fakeGitProvider, repoA, 1) {
		return
	}

	// now lets poll again due to change to the activity to detect the staging is complete
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	assertHasNoPullRequestForEnv(t, activities, a.Name, envNameC)
	assertHasPromoteStatus(t, activities, a.Name, envNameA, v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, envNameB, v1.ActivityStatusTypeRunning)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	if !assertSetPullRequestMerged(t, fakeGitProvider, repoB, 1) {
		return
	}
	if !assertSetPullRequestComplete(t, fakeGitProvider, repoB, 1) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	// C should have started now
	assertHasPullRequestForEnv(t, activities, a.Name, envNameC)
	assertHasPromoteStatus(t, activities, a.Name, envNameA, v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, envNameB, v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, envNameC, v1.ActivityStatusTypeRunning)

	if !assertSetPullRequestMerged(t, fakeGitProvider, repoC, 1) {
		return
	}
	if !assertSetPullRequestComplete(t, fakeGitProvider, repoC, 1) {
		return
	}

	// should be complete now
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	assertHasPromoteStatus(t, activities, a.Name, envNameA, v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, envNameB, v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, envNameC, v1.ActivityStatusTypeSucceeded)

	assertAllPromoteStepsSuccessful(t, activities, a.Name)
}

// TestNewVersionWhileExistingWorkflow lets test that we create a new workflow and terminate
// the old workflow if we find a new version
func TestNewVersionWhileExistingWorkflow(t *testing.T) {
	testOrgName := "jstrachan"
	testRepoName := "myrepo"
	stagingRepoName := "environment-staging"
	prodRepoName := "environment-production"

	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	stagingRepo := gits.NewFakeRepository(testOrgName, stagingRepoName)
	prodRepo := gits.NewFakeRepository(testOrgName, prodRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)

	o := &cmd.ControllerWorkflowOptions{
		NoWatch:          true,
		FakePullRequests: NewCreateEnvPullRequestFn(fakeGitProvider),
		FakeGitProvider:  fakeGitProvider,
	}

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://github.com/"+testOrgName+"/"+stagingRepoName+".git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://github.com/"+testOrgName+"/"+prodRepoName+".git")
	staging.Spec.Order = 100
	production.Spec.Order = 200

	myFlowName := "myflow"

	step1 := workflow.CreateWorkflowPromoteStep("staging")
	step2 := workflow.CreateWorkflowPromoteStep("production", step1)

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			staging,
			production,
			kube.NewPreviewEnvironment("jx-jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment("jx-jstrachan-another-pr-3"),
			workflow.CreateWorkflow("jx", myFlowName,
				step1,
				step2,
			),
		},
		gits.NewGitCLI(),
		helm.NewHelmCLI("helm", helm.V2, ""),
	)
	o.GitClient = &gits.GitFake{}

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)
	if err == nil {
		workflow, err := workflow.GetWorkflow("", jxClient, ns)
		assert.NoError(t, err)
		if err == nil {
			assert.Equal(t, "default", workflow.Name, "name")
			spec := workflow.Spec
			assert.Equal(t, 2, len(spec.Steps), "number of steps")
			if len(spec.Steps) > 0 {
				assertPromoteStep(t, &spec.Steps[0], "staging")
			}
			if len(spec.Steps) > 1 {
				assertPromoteStep(t, &spec.Steps[1], "production")
			}
		}
	}

	a, err := createTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", myFlowName)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	err = o.Run()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	assertHasPullRequestForEnv(t, activities, a.Name, "staging")
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets trigger a new pipeline release which should close the old version
	aOld := a
	a, err = createTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "2", myFlowName)

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	assertHasPullRequestForEnv(t, activities, a.Name, "staging")
	assertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets make sure we don't create a PR for production as we have not completed the staging PR yet
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	assertWorkflowStatus(t, activities, aOld.Name, v1.ActivityStatusTypeAborted)

	// still no PR merged so cannot create a PR for production
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// test no PR on production until staging completed
	if !assertSetPullRequestMerged(t, fakeGitProvider, stagingRepo, 2) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	if !assertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 2) {
		return
	}

	// now lets poll again due to change to the activity to detect the staging is complete
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPullRequestForEnv(t, activities, a.Name, "production")
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	if !assertSetPullRequestMerged(t, fakeGitProvider, prodRepo, 1) {
		return
	}
	if !assertSetPullRequestComplete(t, fakeGitProvider, prodRepo, 1) {
		return
	}

	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeSucceeded)

	assertAllPromoteStepsSuccessful(t, activities, a.Name)
}

func TestPullRequestNumber(t *testing.T) {
	failUrls := []string{"https://github.com/foo/bar/pulls"}
	for _, u := range failUrls {
		_, err := cmd.PullRequestURLToNumber(u)
		assert.Errorf(t, err, "Expected error for pullRequestURLToNumber() with %s", u)
	}

	tests := map[string]int{
		"https://github.com/foo/bar/pulls/12": 12,
	}

	for u, expected := range tests {
		actual, err := cmd.PullRequestURLToNumber(u)
		assert.NoError(t, err, "pullRequestURLToNumber() should not fail for %s", u)
		if err == nil {
			assert.Equal(t, expected, actual, "pullRequestURLToNumber() for %s", u)
		}
	}
}

func dumpPipelineMap(o *cmd.ControllerWorkflowOptions) {
	log.Infof("Dumping PipelineMap {\n")
	for k, v := range o.PipelineMap() {
		log.Infof("    Pipeline %s %s\n", k, v.Name)
	}
	log.Infof("}\n")
}

func pollGitStatusAndReactToPipelineChanges(t *testing.T, o *cmd.ControllerWorkflowOptions, jxClient versioned.Interface, ns string) error {
	o.ReloadAndPollGitPipelineStatuses(jxClient, ns)
	err := o.Run()
	assert.NoError(t, err, "Failed to react to PipelineActivity changes")
	return err
}

func assertSetPullRequestMerged(t *testing.T, provider *gits.FakeProvider, repository *gits.FakeRepository, prNumber int) bool {
	fakePR := repository.PullRequests[prNumber]
	if !assert.NotNil(t, fakePR, "No PullRequest found on repository %s for number #%d", repository.String(), prNumber) {
		return false
	}
	commitLen := len(fakePR.Commits)
	if !assert.True(t, commitLen > 0, "PullRequest #%d on repository %s has no commits", prNumber, repository.String()) {
		return false
	}
	lastFakeCommit := fakePR.Commits[commitLen-1].Commit
	if !assert.NotNil(t, lastFakeCommit, "PullRequest #%d on repository %s last commit status has no commits", prNumber, repository.String()) {
		return false
	}
	sha := lastFakeCommit.SHA
	merged := true
	fakePR.PullRequest.MergeCommitSHA = &sha
	fakePR.PullRequest.Merged = &merged

	log.Infof("PR %s is now merged\n", fakePR.PullRequest.URL)

	// validate the fake git provider concurs
	testGitInfo := &gits.GitRepositoryInfo{
		Organisation: repository.Owner,
		Name:         repository.Name(),
	}
	pr, err := provider.GetPullRequest(repository.Owner, testGitInfo, prNumber)
	assert.NoError(t, err, "Finding PullRequest %d", prNumber)
	return assert.True(t, pr.Merged != nil && *pr.Merged, "Fake PR %d is merged", prNumber)
}

func assertSetPullRequestComplete(t *testing.T, provider *gits.FakeProvider, repository *gits.FakeRepository, prNumber int) bool {
	fakePR := repository.PullRequests[prNumber]
	if !assert.NotNil(t, fakePR, "No PullRequest found on repository %s for number #%d", repository.String(), prNumber) {
		return false
	}

	l := len(fakePR.Commits)
	if l > 0 {
		fakePR.Commits[l-1].Status = gits.CommitSatusSuccess

		// ensure the commit is on the repo r
		lastCommit := fakePR.Commits[l-1]
		if len(repository.Commits) == 0 {
			repository.Commits = append(repository.Commits, lastCommit)
		} else {
			repository.Commits[len(repository.Commits)-1] = lastCommit
		}
		log.Infof("PR %s has commit status success\n", fakePR.PullRequest.URL)
	}

	// validate the fake git provider concurs
	repoOwner := repository.Owner
	repoName := repository.Name()
	testGitInfo := &gits.GitRepositoryInfo{
		Organisation: repoOwner,
		Name:         repoName,
	}
	pr, err := provider.GetPullRequest(repoOwner, testGitInfo, prNumber)
	assert.NoError(t, err, "Finding PullRequest %d", prNumber)
	if !assert.NotNil(t, pr, "Could not find PR %d", prNumber) {
		return false
	}
	if !assert.NotNil(t, pr.MergeCommitSHA, "PR %d has no MergeCommitSHA", prNumber) {
		return false
	}

	statuses, err := provider.ListCommitStatus(repoOwner, repoName, *pr.MergeCommitSHA)
	assert.NoError(t, err, "Finding PullRequest %d commit status", prNumber)
	if assert.True(t, len(statuses) > 0, "PullRequest %d statuses are empty", prNumber) {
		lastStatus := statuses[len(statuses)-1]
		return assert.Equal(t, "success", lastStatus.State, "Last commit status of PullRequest 1 at %s", pr.URL)
	}
	return false
}

func setPullRequestClosed(pr *gits.FakePullRequest) {
	now := time.Now()
	pr.PullRequest.ClosedAt = &now

	log.Infof("PR %s is now closed\n", pr.PullRequest.URL)
}

func assertHasPullRequestForEnv(t *testing.T, activities typev1.PipelineActivityInterface, name string, envName string) {
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		assert.NoError(t, err, "Could not find PipelineActivity %s", name)
		return
	}
	for _, step := range activity.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			if promote.Environment == envName {
				failed := false
				pullRequestStep := promote.PullRequest
				if pullRequestStep == nil {
					assert.Fail(t, "No PullRequest object on PipelineActivity %s for Promote step for Environment %s", name, envName)
					failed = true
				}
				u := pullRequestStep.PullRequestURL
				log.Infof("Found Promote PullRequest %s on PipelineActivity %s for Environment %s\n", u, name, envName)

				if !assert.True(t, u != "", "No PullRequest URL on PipelineActivity %s for Promote step for Environment %s", name, envName) {
					failed = true
				}
				if failed {
					dumpFailedActivity(activity)
				}
				return
			}
		}
	}
	assert.Fail(t, "Missing Promote", "No Promote found on PipelineActivity %s for Environment %s", name, envName)
	dumpFailedActivity(activity)
}

func assertHasPromoteStatus(t *testing.T, activities typev1.PipelineActivityInterface, name string, envName string, status v1.ActivityStatusType) {
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		assert.NoError(t, err, "Could not find PipelineActivity %s", name)
		return
	}
	for _, step := range activity.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			if promote.Environment == envName {
				if !assert.Equal(t, string(status), string(promote.Status), "activity status for %s promote %s", name, envName) {
					dumpFailedActivity(activity)
				}
				return
			}
		}
	}
	assert.Fail(t, "Missing Promote", "No Promote found on PipelineActivity %s for Environment %s", name, envName)
	dumpFailedActivity(activity)
}

func assertHasPipelineStatus(t *testing.T, activities typev1.PipelineActivityInterface, name string, status v1.ActivityStatusType) {
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		assert.NoError(t, err, "Could not find PipelineActivity %s", name)
		return
	}
	if !assert.Equal(t, string(status), string(activity.Spec.Status), "activity status for PipelineActivity %s", name) {
		dumpFailedActivity(activity)
	}
}

func assertAllPromoteStepsSuccessful(t *testing.T, activities typev1.PipelineActivityInterface, name string) {
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		assert.NoError(t, err, "Could not find PipelineActivity %s", name)
		return
	}
	assert.Equal(t, string(v1.ActivityStatusTypeSucceeded), string(activity.Spec.Status), "PipelineActivity status for %s", activity.Name)
	assert.Equal(t, string(v1.ActivityStatusTypeSucceeded), string(activity.Spec.WorkflowStatus), "PipelineActivity workflow status for %s", activity.Name)
	for _, step := range activity.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			assert.Equal(t, string(v1.ActivityStatusTypeSucceeded), string(promote.Status), "PipelineActivity %s status for Promote to Environment %s", activity.Name, promote.Environment)
		}
	}
}

func assertWorkflowStatus(t *testing.T, activities typev1.PipelineActivityInterface, name string, status v1.ActivityStatusType) {
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		assert.NoError(t, err, "Could not find PipelineActivity %s", name)
		return
	}
	if !assert.Equal(t, string(status), string(activity.Spec.Status), "PipelineActivity status for %s", activity.Name) ||
		!assert.Equal(t, string(status), string(activity.Spec.WorkflowStatus), "PipelineActivity workflow status for %s", activity.Name) {
		dumpFailedActivity(activity)
	}
}

func dumpFailedActivity(activity *v1.PipelineActivity) {
	data, err := yaml.Marshal(activity)
	if err == nil {
		log.Warnf("YAML: %s\n", string(data))
	}
}

func assertHasNoPullRequestForEnv(t *testing.T, activities typev1.PipelineActivityInterface, name string, envName string) {
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		assert.NoError(t, err, "Could not find PipelineActivity %s", name)
		return
	}
	for _, step := range activity.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			if promote.Environment == envName {
				assert.Fail(t, "Should not have a Promote for Environment %s but has %v", envName, promote)
				return
			}
		}
	}
}

func createTestPipelineActivity(jxClient versioned.Interface, ns string, folder string, repo string, branch string, build string, workflow string) (*v1.PipelineActivity, error) {
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	key := &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:     folder + "-" + repo + "-" + branch + "-" + build,
			Pipeline: folder + "/" + repo + "/" + branch,
			Build:    build,
		},
	}
	a, _, err := key.GetOrCreate(activities)
	version := "1.0." + build
	a.Spec.GitOwner = folder
	a.Spec.GitRepository = repo
	a.Spec.GitURL = "https://github.com/" + folder + "/" + repo + ".git"
	a.Spec.Version = version
	a.Spec.Workflow = workflow
	_, err = activities.Update(a)
	return a, err
}

func CreateFakePullRequest(repository *gits.FakeRepository, env *v1.Environment, modifyRequirementsFn cmd.ModifyRequirementsFn, branchNameText string, title string, message string, pullRequestInfo *cmd.ReleasePullRequestInfo) (*cmd.ReleasePullRequestInfo, error) {
	if pullRequestInfo == nil {
		pullRequestInfo = &cmd.ReleasePullRequestInfo{}
	}

	if pullRequestInfo.PullRequest == nil {
		pullRequestInfo.PullRequest = &gits.GitPullRequest{}
	}
	pr := pullRequestInfo.PullRequest
	if pr.Number == nil {
		repository.PullRequestCounter++
		n := repository.PullRequestCounter
		pr.Number = &n
	}
	if pr.URL == "" {
		n := *pr.Number
		pr.URL = "https://github.com/" + repository.Owner + "/" + repository.Name() + "/pulls/" + strconv.Itoa(n)
	}
	if pr.Owner == "" {
		pr.Owner = repository.Owner
	}
	if pr.Repo == "" {
		pr.Repo = repository.Name()
	}

	log.Infof("Creating fake Pull Request for env %s branch %s title %s message %s with number %d and URL %s\n", env.Name, branchNameText, title, message, *pr.Number, pr.URL)

	if pr != nil && pr.Number != nil {
		n := *pr.Number
		log.Infof("Creating fake PullRequest number %d at URL %s\n", n, pr.URL)

		// lets add a pending commit too
		commitSha := string(uuid.NewUUID())
		commit := &gits.FakeCommit{
			Commit: &gits.GitCommit{
				SHA:     commitSha,
				Message: "dummy commit " + commitSha,
			},
			Status: gits.CommitStatusPending,
		}

		repository.PullRequests[n] = &gits.FakePullRequest{
			PullRequest: pr,
			Commits:     []*gits.FakeCommit{commit},
			Comment:     "comment for PR",
		}
		repository.Commits = append(repository.Commits, commit)
	} else {
		log.Warnf("Missing number for PR %s\n", pr.URL)
	}
	return pullRequestInfo, nil
}

func NewCreateEnvPullRequestFn(provider *gits.FakeProvider) cmd.CreateEnvPullRequestFn {
	fakePrFn := func(env *v1.Environment, modifyRequirementsFn cmd.ModifyRequirementsFn, branchNameText string, title string, message string, pullRequestInfo *cmd.ReleasePullRequestInfo) (*cmd.ReleasePullRequestInfo, error) {
		envURL := env.Spec.Source.URL
		values := []string{}
		for _, repos := range provider.Repositories {
			for _, repo := range repos {
				cloneURL := repo.GitRepo.CloneURL
				if cloneURL == envURL {
					return CreateFakePullRequest(repo, env, modifyRequirementsFn, branchNameText, title, message, pullRequestInfo)
				}
				values = append(values, cloneURL)
			}
		}
		return nil, fmt.Errorf("Could not find repository for cloneURL %s values found %s", envURL, strings.Join(values, ", "))
	}
	return fakePrFn
}
