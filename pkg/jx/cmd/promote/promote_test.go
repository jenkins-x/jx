package promote_test

import (
	"encoding/json"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jx/cmd/controller"
	"github.com/jenkins-x/jx/pkg/jx/cmd/promote"
	"github.com/jenkins-x/jx/pkg/jx/cmd/testhelpers"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/petergtz/pegomock"

	"k8s.io/helm/pkg/proto/hapi/chart"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
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

	promoteOptions := &promote.PromoteOptions{
		Environment:          "production",                   // --env production
		Application:          "my-app",                       // --app my-app
		Pipeline:             testEnv.Activity.Spec.Pipeline, // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Build:                testEnv.Activity.Spec.Build,    // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Version:              version,                        // --version 1.2.0
		ReleaseName:          "",
		LocalHelmRepoName:    "",
		HelmRepositoryURL:    "",
		NoHelmUpdate:         true, // --no-helm-update
		AllAutomatic:         false,
		NoMergePullRequest:   false,
		NoPoll:               true, // --no-poll
		NoWaitAfterMerge:     false,
		IgnoreLocalFiles:     true,
		Timeout:              "1h",
		PullRequestPollTime:  "20s",
		Filter:               "",
		Alias:                "",
		Namespace:            "jx",
		ConfigureGitCallback: testEnv.ConfigureGitFolderFn,
	}
	commonOpts := *testEnv.CommonOptions
	promoteOptions.CommonOptions = &commonOpts // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.BatchMode = true            // --batch-mode

	// Check there is no PR for production env yet
	jxClient, ns, err := promoteOptions.JXClientAndDevNamespace()
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	testhelpers.AssertHasNoPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")

	// Run the promotion
	err = promoteOptions.Run()
	assert.NoError(t, err)

	// The PR has been created
	testhelpers.AssertHasPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")
	testhelpers.AssertHasPipelineStatus(t, activities, testEnv.Activity.Name, v1.ActivityStatusTypeRunning)
	// merge
	testhelpers.AssertSetPullRequestMerged(t, testEnv.FakeGitProvider, testEnv.ProdRepo.Owner, testEnv.ProdRepo.Name(), 1)
	testhelpers.AssertSetPullRequestComplete(t, testEnv.FakeGitProvider, testEnv.ProdRepo, 1)

	// retry the workflow to actually check the PR was merged and the app is in production
	pollGitStatusAndReactToPipelineChanges(t, testEnv.WorkflowOptions, jxClient, ns)
	testhelpers.AssertHasPromoteStatus(t, activities, testEnv.Activity.Name, "production", v1.ActivityStatusTypeSucceeded)
	assert.Equal(t, version, promoteOptions.ReleaseInfo.Version)

}

func TestPromoteToProductionNoMergeRun(t *testing.T) {

	// prepare the initial setup for testing
	testEnv, err := prepareInitialPromotionEnv(t, true)
	assert.NoError(t, err)

	// jx promote --batch-mode --app my-app --env production --no-merge --no-helm-update

	promoteOptions := &promote.PromoteOptions{
		Environment:          "production",                   // --env production
		Application:          "my-app",                       // --app my-app
		Pipeline:             testEnv.Activity.Spec.Pipeline, // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Build:                testEnv.Activity.Spec.Build,    // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Version:              "",
		ReleaseName:          "",
		LocalHelmRepoName:    "",
		HelmRepositoryURL:    "",
		NoHelmUpdate:         true, // --no-helm-update
		AllAutomatic:         false,
		NoMergePullRequest:   true,  // --no-merge
		NoPoll:               false, // note polling enabled
		NoWaitAfterMerge:     false,
		IgnoreLocalFiles:     true,
		Timeout:              "1h",
		PullRequestPollTime:  "20s",
		Filter:               "",
		Alias:                "",
		Namespace:            "jx",
		ConfigureGitCallback: testEnv.ConfigureGitFolderFn,
	}

	commonOpts := *testEnv.CommonOptions
	promoteOptions.CommonOptions = &commonOpts // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.BatchMode = true            // --batch-mode

	jxClient, ns, err := promoteOptions.JXClientAndDevNamespace()
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	testhelpers.AssertHasNoPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")

	ch := make(chan int)

	// run the promote command in parallel
	go func() {
		err = promoteOptions.Run()
		assert.NoError(t, err)
		close(ch)
	}()

	// wait for the PR the be created by the promote command
	testhelpers.WaitForPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")
	testhelpers.AssertHasPipelineStatus(t, activities, testEnv.Activity.Name, v1.ActivityStatusTypeRunning)

	// merge the PR created by promote command...
	testhelpers.AssertSetPullRequestMerged(t, testEnv.FakeGitProvider, testEnv.ProdRepo.Owner, testEnv.ProdRepo.Name(), 1)
	testhelpers.AssertSetPullRequestComplete(t, testEnv.FakeGitProvider, testEnv.ProdRepo, 1)

	// ...and wait for the Run routine to finish (it was polling on the PR to be merged)
	<-ch

	// retry the workflow to actually check the PR was merged and the app is in production
	pollGitStatusAndReactToPipelineChanges(t, testEnv.WorkflowOptions, jxClient, ns)
	testhelpers.AssertHasPromoteStatus(t, activities, testEnv.Activity.Name, "production", v1.ActivityStatusTypeSucceeded)

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

	promoteOptions := &promote.PromoteOptions{
		Environment:          "production",                   // --env production
		Application:          "my-app",                       // --app my-app
		Pipeline:             testEnv.Activity.Spec.Pipeline, // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Build:                testEnv.Activity.Spec.Build,    // needed for the test to pass on CI, otherwise it takes the actual CI build value
		Version:              "",
		ReleaseName:          "",
		LocalHelmRepoName:    "",
		HelmRepositoryURL:    "",
		NoHelmUpdate:         true, // --no-helm-update
		AllAutomatic:         false,
		NoMergePullRequest:   false, // note auto-merge enabled
		NoPoll:               false, // note polling enabled
		NoWaitAfterMerge:     false,
		IgnoreLocalFiles:     true,
		Timeout:              "1h",
		PullRequestPollTime:  "20s",
		Filter:               "",
		Alias:                "",
		Namespace:            "jx",
		ConfigureGitCallback: testEnv.ConfigureGitFolderFn,
	}

	commonOpts := *testEnv.CommonOptions
	promoteOptions.CommonOptions = &commonOpts // Factory and other mocks initialized by cmd.ConfigureTestOptionsWithResources
	promoteOptions.BatchMode = true            // --batch-mode

	jxClient, ns, err := promoteOptions.JXClientAndDevNamespace()
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	testhelpers.AssertHasNoPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")

	ch := make(chan int)

	// run the promote command in parallel
	go func() {
		err = promoteOptions.Run()
		assert.NoError(t, err)
		close(ch)
	}()

	// wait for the PR the be created by the promote command
	testhelpers.WaitForPullRequestForEnv(t, activities, testEnv.Activity.Name, "production")
	testhelpers.AssertHasPipelineStatus(t, activities, testEnv.Activity.Name, v1.ActivityStatusTypeRunning)

	// mark latest commit as success tu unblock the promotion (PR will be automatically merged)
	testhelpers.SetSuccessCommitStatusInPR(t, testEnv.ProdRepo, 1)

	// ...and wait for the Run routine to finish (it was polling on the PR last commit status success to auto-merge)
	<-ch

	// retry the workflow to actually check the PR was merged and the app is in production
	pollGitStatusAndReactToPipelineChanges(t, testEnv.WorkflowOptions, jxClient, ns)
	testhelpers.AssertHasPromoteStatus(t, activities, testEnv.Activity.Name, "production", v1.ActivityStatusTypeSucceeded)

	//TODO: promoteOptions.ReleaseInfo.Version is empty here. Is this a bug?
	//assert.Equal(t, "1.0.1", promoteOptions.ReleaseInfo.Version) // default next version

	// however it looks like the activity contains the correct version...
	assert.Equal(t, "1.0.1", testEnv.Activity.Spec.Version)
}

// Contains all useful data from the test environment initialized by `prepareInitialPromotionEnv`
type TestEnv struct {
	Activity             *v1.PipelineActivity
	WorkflowOptions      *controller.ControllerWorkflowOptions
	CommonOptions        *opts.CommonOptions
	FakeGitProvider      *gits.FakeProvider
	DevRepo              *gits.FakeRepository
	StagingRepo          *gits.FakeRepository
	ProdRepo             *gits.FakeRepository
	ConfigureGitFolderFn gits.ConfigureGitFn
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

	// Needed for another helpe

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)
	fakeGitProvider.User.Username = testOrgName

	o := &controller.ControllerWorkflowOptions{
		CommonOptions: &opts.CommonOptions{},
		NoWatch:       true,
		Namespace:     "jx",
	}

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://fake.git/"+testOrgName+"/"+stagingRepoName+"."+
		"git")
	production := kube.NewPermanentEnvironmentWithGit("production",
		"https://fake.git/"+testOrgName+"/"+prodRepoName+".git")
	if productionManualPromotion {
		production.Spec.PromotionStrategy = v1.PromotionStrategyTypeManual
	}

	err := testhelpers.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)
	configureGitFn := func(dir string, gitInfo *gits.GitRepository, gitter gits.Gitter) error {
		err := gitter.Init(dir)
		if err != nil {
			return err
		}
		// Really we should have a dummy environment chart but for now let's just mock it out as needed
		err = os.MkdirAll(filepath.Join(dir, "templates"), 0700)
		if err != nil {
			return err
		}
		data, err := json.Marshal(staging)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, "templates", "environment-staging.yaml"), data, 0755)
		if err != nil {
			return err
		}
		data, err = json.Marshal(production)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, "templates", "environment-production.yaml"), data, 0755)
		if err != nil {
			return err
		}
		return gitter.AddCommit(dir, "Initial Commit")
	}

	o.ConfigureGitFn = configureGitFn

	workflowName := "default"

	mockHelmer := helm_test.NewMockHelmer()
	testhelpers.ConfigureTestOptionsWithResources(o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			staging,
			production,
			kube.NewPreviewEnvironment("preview-pr-1"),
		},
		gits.NewGitLocal(),
		fakeGitProvider,
		mockHelmer,
		resources_mock.NewMockInstaller(),
	)

	//Mock out the helm repository fetch operation
	helm_test.StubFetchChart(testRepoName, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    testRepoName,
			Version: "1.0.1",
		},
	}, mockHelmer)

	// Mock out the search versions operation

	pegomock.When(mockHelmer.SearchChartVersions(testRepoName)).ThenReturn([]string{"1.0.1"}, nil)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)

	a, err := testhelpers.CreateTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", workflowName)
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
	testhelpers.AssertHasPullRequestForEnv(t, activities, a.Name, "staging")
	testhelpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// react to the new PR in staging
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	// lets make sure we don't create a PR for production as its manual
	testhelpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// merge PR in staging repo
	if !testhelpers.AssertSetPullRequestMerged(t, fakeGitProvider, stagingRepo.Owner, stagingRepo.Name(), 1) {
		return nil, err
	}
	if !testhelpers.AssertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 1) {
		return nil, err
	}

	// react to the PR merge in staging
	pollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	// the pipeline activity succeeded
	testhelpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	// There is no PR for production, as it is manual
	testhelpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// Promote to staging succeeded...
	testhelpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	// ...and all promote-to-staging steps were successful
	testhelpers.AssertAllPromoteStepsSuccessful(t, activities, a.Name)

	return &TestEnv{
		Activity:             a,
		CommonOptions:        o.CommonOptions,
		WorkflowOptions:      o,
		FakeGitProvider:      fakeGitProvider,
		DevRepo:              fakeRepo,
		StagingRepo:          stagingRepo,
		ProdRepo:             prodRepo,
		ConfigureGitFolderFn: configureGitFn,
	}, nil
}

func pollGitStatusAndReactToPipelineChanges(t *testing.T, o *controller.ControllerWorkflowOptions, jxClient versioned.Interface, ns string) error {
	o.ReloadAndPollGitPipelineStatuses(jxClient, ns)
	err := o.Run()
	assert.NoError(t, err, "Failed to react to PipelineActivity changes")
	return err
}
