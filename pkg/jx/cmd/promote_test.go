package cmd_test

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/runtime"

)

func TestPromoteToProductionRun(t *testing.T) {

	// prepare the initial setup for testing
	testEnv, err := prepareInitialPromotionEnv(t, true)
	assert.NoError(t, err)

	// jx promote --batch-mode --app my-app --env production --version 1.2.0 --no-helm-update --no-poll

	version := "1.2.0"

	o := &cmd.PromoteOptions{
		Environment:        "production", // --env production
		Application:        "my-app", // --app my-app
		Pipeline:           "",
		Build:              "",
		Version:            version, // --version 1.2.0
		ReleaseName:        "",
		LocalHelmRepoName:  "",
		HelmRepositoryURL:  "",
		NoHelmUpdate:       true, // --no-helm-update
		AllAutomatic:       false,
		NoMergePullRequest: false,
		NoPoll:             true, // to test "false" here provider_fake#UpdatePullRequestStatus needs to be implemented
		NoWaitAfterMerge:   false,
		IgnoreLocalFiles:   true,
		Timeout:            "1h",
		PullRequestPollTime:"20s",
		Filter:             "",
		Alias:              "",
		FakePullRequests:	testEnv.FakePullRequests,

		// test settings
		UseFakeHelm: true,
	}
	o.CommonOptions = *testEnv.CommonOptions // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	o.BatchMode = true // --batch-mode

	log.Infof("Promoting to production version %s for app my-app\n", version)
	err = o.Run()
	assert.NoError(t, err)

	assert.Equal(t, o.ReleaseInfo.Version, version)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	cmd.AssertHasPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")
	cmd.AssertHasPipelineStatus(t, activities, testEnv.Activity.Name, v1.ActivityStatusTypeRunning)

	cmd.AssertSetPullRequestMerged(t, testEnv.FakeGitProvider, testEnv.ProdRepo, 1)
	cmd.AssertSetPullRequestComplete(t, testEnv.FakeGitProvider, testEnv.ProdRepo, 1)

	// retry the workflow to actually check the PR was merged and the app is in production
	cmd.PollGitStatusAndReactToPipelineChanges(t, testEnv.WorkflowOptions, jxClient, ns)
	cmd.AssertHasPromoteStatus(t, activities, testEnv.Activity.Name, "production", v1.ActivityStatusTypeSucceeded)

}

// Contains all useful data from the test environment initialized by `prepareInitialPromotionEnv`
type TestEnv struct {
	Activity *v1.PipelineActivity
	FakePullRequests cmd.CreateEnvPullRequestFn
	WorkflowOptions *cmd.ControllerWorkflowOptions
	CommonOptions *cmd.CommonOptions
	FakeGitProvider *gits.FakeProvider
	DevRepo *gits.FakeRepository
	StagingRepo *gits.FakeRepository
	ProdRepo *gits.FakeRepository
}

// Prepares an initial configuration with a typical environment setup.
// After a call to this function, version 1.0.0 of my-app is in staging, waiting to be promoted to production.
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
	}

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://github.com/"+testOrgName+"/"+stagingRepoName+".git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://github.com/"+testOrgName+"/"+prodRepoName+".git")
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
		helm_test.NewMockHelmer(),
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
		Activity: a,
		FakePullRequests: o.FakePullRequests,
		CommonOptions: &o.CommonOptions,
		WorkflowOptions: o,
		FakeGitProvider: fakeGitProvider,
		DevRepo: fakeRepo,
		StagingRepo: stagingRepo,
		ProdRepo: prodRepo,
	}, nil
}