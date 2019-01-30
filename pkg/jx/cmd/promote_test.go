package cmd_test

import (
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/gits"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"

	resources_mock "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestPromoteToProductionRun(t *testing.T) {

	// prepare the initial setup for testing
	testEnv, err := prepareInitialPromotionEnv(t, true)
	assert.NoError(t, err)

	// jx promote --batch-mode --app my-app --env production --version 1.2.0 --no-helm-update --no-poll

	version := "1.2.0"

	promoteOptions := &cmd.PromoteOptions{
		Environment:         "production",                   // --env production
		Application:         "my-app",                       // --app my-app
		Pipeline:            testEnv.Activity.Spec.Pipeline, // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Build:               testEnv.Activity.Spec.Build,    // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Version:             version,                        // --version 1.2.0
		ReleaseName:         "",
		LocalHelmRepoName:   "",
		HelmRepositoryURL:   "",
		NoHelmUpdate:        true, // --no-helm-update
		AllAutomatic:        false,
		NoMergePullRequest:  false,
		NoPoll:              true, // --no-poll
		NoWaitAfterMerge:    false,
		IgnoreLocalFiles:    true,
		Timeout:             "1h",
		PullRequestPollTime: "20s",
		Filter:              "",
		Alias:               "",
		FakePullRequests:    testEnv.FakePullRequests,
		Namespace:           "jx",

		// test settings
		UseFakeHelm: true,
	}
	promoteOptions.CommonOptions = *testEnv.CommonOptions // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.BatchMode = true                       // --batch-mode

	// Check there is no PR for production env yet
	jxClient, ns, err := promoteOptions.JXClientAndDevNamespace()
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	cmd.AssertHasNoPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")

	// Run the promotion
	err = promoteOptions.Run()
	assert.NoError(t, err)

	// The PR has been created
	cmd.AssertHasPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")
	cmd.AssertHasPipelineStatus(t, activities, testEnv.Activity.Name, v1.ActivityStatusTypeRunning)
	// merge
	cmd.AssertSetPullRequestMerged(t, testEnv.FakeGitProvider, testEnv.ProdRepo, 1)
	cmd.AssertSetPullRequestComplete(t, testEnv.FakeGitProvider, testEnv.ProdRepo, 1)

	// retry the workflow to actually check the PR was merged and the app is in production
	cmd.PollGitStatusAndReactToPipelineChanges(t, testEnv.WorkflowOptions, jxClient, ns)
	cmd.AssertHasPromoteStatus(t, activities, testEnv.Activity.Name, "production", v1.ActivityStatusTypeSucceeded)
	assert.Equal(t, version, promoteOptions.ReleaseInfo.Version)

}

func TestPromoteToProductionNoMergeRun(t *testing.T) {

	// prepare the initial setup for testing
	testEnv, err := prepareInitialPromotionEnv(t, true)
	assert.NoError(t, err)

	// jx promote --batch-mode --app my-app --env production --no-merge --no-helm-update

	promoteOptions := &cmd.PromoteOptions{
		Environment:         "production",                   // --env production
		Application:         "my-app",                       // --app my-app
		Pipeline:            testEnv.Activity.Spec.Pipeline, // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Build:               testEnv.Activity.Spec.Build,    // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Version:             "",
		ReleaseName:         "",
		LocalHelmRepoName:   "",
		HelmRepositoryURL:   "",
		NoHelmUpdate:        true, // --no-helm-update
		AllAutomatic:        false,
		NoMergePullRequest:  true,  // --no-merge
		NoPoll:              false, // note polling enabled
		NoWaitAfterMerge:    false,
		IgnoreLocalFiles:    true,
		Timeout:             "1h",
		PullRequestPollTime: "20s",
		Filter:              "",
		Alias:               "",
		FakePullRequests:    testEnv.FakePullRequests,
		Namespace:           "jx",

		// test settings
		UseFakeHelm: true,
	}

	promoteOptions.CommonOptions = *testEnv.CommonOptions // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.BatchMode = true                       // --batch-mode

	jxClient, ns, err := promoteOptions.JXClientAndDevNamespace()
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	cmd.AssertHasNoPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")

	ch := make(chan int)

	// run the promote command in parallel
	go func() {
		err = promoteOptions.Run()
		assert.NoError(t, err)
		close(ch)
	}()

	// wait for the PR the be created by the promote command
	cmd.WaitForPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")
	cmd.AssertHasPipelineStatus(t, activities, testEnv.Activity.Name, v1.ActivityStatusTypeRunning)

	// merge the PR created by promote command...
	cmd.AssertSetPullRequestMerged(t, testEnv.FakeGitProvider, testEnv.ProdRepo, 1)
	cmd.AssertSetPullRequestComplete(t, testEnv.FakeGitProvider, testEnv.ProdRepo, 1)

	// ...and wait for the Run routine to finish (it was polling on the PR to be merged)
	<-ch

	// retry the workflow to actually check the PR was merged and the app is in production
	cmd.PollGitStatusAndReactToPipelineChanges(t, testEnv.WorkflowOptions, jxClient, ns)
	cmd.AssertHasPromoteStatus(t, activities, testEnv.Activity.Name, "production", v1.ActivityStatusTypeSucceeded)

	//TODO: promoteOptions.ReleaseInfo.Version is empty here. Is this a bug?
	//assert.Equal(t, "1.0.1", promoteOptions.ReleaseInfo.Version) // default next version

	// however it looks like the activity contains the correct version...
	assert.Equal(t, "1.0.1", testEnv.Activity.Spec.Version)
}

func TestPromoteToProductionPRPollingRun(t *testing.T) {

	// prepare the initial setup for testing
	testEnv, err := prepareInitialPromotionEnv(t, true)
	assert.NoError(t, err)

	// jx promote --batch-mode --app my-app --env production --no-helm-update

	promoteOptions := &cmd.PromoteOptions{
		Environment:         "production",                   // --env production
		Application:         "my-app",                       // --app my-app
		Pipeline:            testEnv.Activity.Spec.Pipeline, // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Build:               testEnv.Activity.Spec.Build,    // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Version:             "",
		ReleaseName:         "",
		LocalHelmRepoName:   "",
		HelmRepositoryURL:   "",
		NoHelmUpdate:        true, // --no-helm-update
		AllAutomatic:        false,
		NoMergePullRequest:  false, // note auto-merge enabled
		NoPoll:              false, // note polling enabled
		NoWaitAfterMerge:    false,
		IgnoreLocalFiles:    true,
		Timeout:             "1h",
		PullRequestPollTime: "20s",
		Filter:              "",
		Alias:               "",
		FakePullRequests:    testEnv.FakePullRequests,
		Namespace:           "jx",
		// test settings
		UseFakeHelm: true,
	}

	promoteOptions.CommonOptions = *testEnv.CommonOptions // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.BatchMode = true                       // --batch-mode

	jxClient, ns, err := promoteOptions.JXClientAndDevNamespace()
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	cmd.AssertHasNoPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")

	ch := make(chan int)

	// run the promote command in parallel
	go func() {
		err = promoteOptions.Run()
		assert.NoError(t, err)
		close(ch)
	}()

	// wait for the PR the be created by the promote command
	cmd.WaitForPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")
	cmd.AssertHasPipelineStatus(t, activities, testEnv.Activity.Name, v1.ActivityStatusTypeRunning)

	// mark latest commit as success tu unblock the promotion (PR will be automatically merged)
	cmd.SetSuccessCommitStatusInPR(t, testEnv.ProdRepo, 1)

	// ...and wait for the Run routine to finish (it was polling on the PR last commit status success to auto-merge)
	<-ch

	// retry the workflow to actually check the PR was merged and the app is in production
	cmd.PollGitStatusAndReactToPipelineChanges(t, testEnv.WorkflowOptions, jxClient, ns)
	cmd.AssertHasPromoteStatus(t, activities, testEnv.Activity.Name, "production", v1.ActivityStatusTypeSucceeded)

	//TODO: promoteOptions.ReleaseInfo.Version is empty here. Is this a bug?
	//assert.Equal(t, "1.0.1", promoteOptions.ReleaseInfo.Version) // default next version

	// however it looks like the activity contains the correct version...
	assert.Equal(t, "1.0.1", testEnv.Activity.Spec.Version)
}

// Contains all useful data from the test environment initialized by `prepareInitialPromotionEnv`
type TestEnv struct {
	Activity         *v1.PipelineActivity
	FakePullRequests cmd.CreateEnvPullRequestFn
	WorkflowOptions  *cmd.ControllerWorkflowOptions
	CommonOptions    *cmd.CommonOptions
	FakeGitProvider  *gits.FakeProvider
	DevRepo          *gits.FakeRepository
	StagingRepo      *gits.FakeRepository
	ProdRepo         *gits.FakeRepository
}

// Prepares an initial configuration with a typical environment setup.
// After a call to this function, version 1.0.1 of my-app is in staging, waiting to be promoted to production.
// It also prepare fakes of kube, jxClient, etc.
func prepareInitialPromotionEnv(t *testing.T, productionManualPromotion bool) (*TestEnv, error) {
	testOrgName := "myorg"
	testRepoName := "my-app"
	stagingRepoName := "environment-staging"
	prodRepoName := "environment-production"

	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	stagingRepo := gits.NewFakeRepository(testOrgName, stagingRepoName)
	prodRepo := gits.NewFakeRepository(testOrgName, prodRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)

	o := &cmd.ControllerWorkflowOptions{
		NoWatch:          true,
		FakePullRequests: cmd.NewCreateEnvPullRequestFn(fakeGitProvider),
		FakeGitProvider:  fakeGitProvider,
		Namespace:        "jx",
	}

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://fake.git/"+testOrgName+"/"+stagingRepoName+"."+
		"git")
	production := kube.NewPermanentEnvironmentWithGit("production",
		"https://fake.git/"+testOrgName+"/"+prodRepoName+".git")
	if productionManualPromotion {
		production.Spec.PromotionStrategy = v1.PromotionStrategyTypeManual
	}

	workflowName := "default"

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			staging,
			production,
			kube.NewPreviewEnvironment("preview-pr-1"),
		},
		&gits.GitFake{},
		nil,
		helm_test.NewMockHelmer(),
		resources_mock.NewMockInstaller(),
	)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)

	a, err := cmd.CreateTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", workflowName)
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}

	err = o.Run()
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	cmd.AssertHasPullRequestForEnv(t, activities, a.Name, "staging")
	cmd.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// react to the new PR in staging
	cmd.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	// lets make sure we don't create a PR for production as its manual
	cmd.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// merge PR in staging repo
	if !cmd.AssertSetPullRequestMerged(t, fakeGitProvider, stagingRepo, 1) {
		return nil, err
	}
	if !cmd.AssertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 1) {
		return nil, err
	}

	// react to the PR merge in staging
	cmd.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	// the pipeline activity succeeded
	cmd.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	// There is no PR for production, as it is manual
	cmd.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// Promote to staging suceeded...
	cmd.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	// ...and all promote-to-staging steps were successful
	cmd.AssertAllPromoteStepsSuccessful(t, activities, a.Name)

	return &TestEnv{
		Activity:         a,
		FakePullRequests: o.FakePullRequests,
		CommonOptions:    &o.CommonOptions,
		WorkflowOptions:  o,
		FakeGitProvider:  fakeGitProvider,
		DevRepo:          fakeRepo,
		StagingRepo:      stagingRepo,
		ProdRepo:         prodRepo,
	}, nil
}
