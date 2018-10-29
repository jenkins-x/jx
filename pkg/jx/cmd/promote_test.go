package cmd_test

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/runtime"

)

func TestPromoteRun(t *testing.T) {

	a, fakePullRequests, commonOptions, err := prepareInitialPromotionEnv(t, true)
	assert.NoError(t, err)

	// jx promote -b --timeout 1h --version 1.0.0 --no-helm-update

	o := &cmd.PromoteOptions{
		Namespace:          "production",
		Environment:        "production",
		Application:        "my-app",
		Pipeline:           "",
		Build:              "",
		Version:            "1.0.0", // --version 1.0.0
		ReleaseName:        "",
		LocalHelmRepoName:  "",
		HelmRepositoryURL:  "",
		NoHelmUpdate:       true, // --no-helm-update
		AllAutomatic:       true, // --all-auto
		NoMergePullRequest: false,
		NoPoll:             true, // this needs provider_fake#UpdatePullRequestStatus to be implemented
		NoWaitAfterMerge:   false,
		IgnoreLocalFiles:   true,
		Timeout:            "1h", // --timeout 1h
		PullRequestPollTime:"20s",
		Filter:             "",
		Alias:              "",
		FakePullRequests:	*fakePullRequests,
		CommonOptions: 		*commonOptions, // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources

		// test settings
		UseFakeHelm: true,
	}

	log.Infof("Promoting to production version 1.0.0 for app my-app\n",)
	err = o.Run()
	assert.NoError(t, err)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	cmd.AssertHasPullRequestForEnv(t, activities, a.Name, "production")
	cmd.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// more checks

	assert.Equal(t, o.ReleaseInfo.Version, "1.0.0")

}

// Prepares an initial configuration with a typical environment setup
func prepareInitialPromotionEnv(t *testing.T, productionManualPromotion bool) (*v1.PipelineActivity, *cmd.CreateEnvPullRequestFn, *cmd.CommonOptions, error) {
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
		return nil, nil, nil, err
	}

	err = o.Run()
	assert.NoError(t, err)
	if err != nil {
		return nil, nil, nil, err
	}
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	cmd.AssertHasPullRequestForEnv(t, activities, a.Name, "staging")
	cmd.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// react to the new PR in staging
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	// lets make sure we don't create a PR for production as its manual
	cmd.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// merge PR in staging repo
	if !cmd.AssertSetPullRequestMerged(t, fakeGitProvider, stagingRepo, 1) {
		return nil, nil, nil, err
	}
	if !cmd.AssertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 1) {
		return nil, nil, nil, err
	}

	// react to the PR merge in staging
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	// the pipeline activity succeeded
	cmd.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	// There is no PR for production, as it is manual
	cmd.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// Promote to staging suceeded...
	cmd.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	// ...and all promote steps were successful
	cmd.AssertAllPromoteStepsSuccessful(t, activities, a.Name)

	return a, &o.FakePullRequests, &o.CommonOptions, nil
}

func pollGitStatusAndReactToPipelineChanges(t *testing.T, o *cmd.ControllerWorkflowOptions, jxClient versioned.Interface, ns string) error {
	o.ReloadAndPollGitPipelineStatuses(jxClient, ns)
	err := o.Run()
	assert.NoError(t, err, "Failed to react to PipelineActivity changes")
	return err
}