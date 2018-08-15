package cmd

import (
	"strconv"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	testOrgName  = "jstrachan"
	testRepoName = "myrepo"
	testMergeSha = "dummyCommitSha"
)

func TestPullRequestNumber(t *testing.T) {
	failUrls := []string{"https://github.com/foo/bar/pulls"}
	for _, u := range failUrls {
		_, err := pullRequestURLToNumber(u)
		assert.Errorf(t, err, "Expected error for pullRequestURLToNumber() with %s", u)
	}

	tests := map[string]int{
		"https://github.com/foo/bar/pulls/12": 12,
	}

	for u, expected := range tests {
		actual, err := pullRequestURLToNumber(u)
		assert.NoError(t, err, "pullRequestURLToNumber() should not fail for %s", u)
		if err == nil {
			assert.Equal(t, expected, actual, "pullRequestURLToNumber() for %s", u)
		}
	}
}

func TestSequentialWorkflow(t *testing.T) {
	prMap := map[int]*gits.FakePullRequest{}

	fakePrFn := func(env *v1.Environment, modifyRequirementsFn ModifyRequirementsFn, branchNameText string, title string, message string, pullRequestInfo *ReleasePullRequestInfo) (*ReleasePullRequestInfo, error) {
		i, err := createFakePullRequests(env, modifyRequirementsFn, branchNameText, title, message, pullRequestInfo)
		if err == nil {
			if i.PullRequest == nil {
				i.PullRequest = &gits.GitPullRequest{
					Number: &fakePrCounter,
					URL:    createFakePRURL(fakePrCounter),
					Owner:  testOrgName,
					Repo:   testRepoName,
				}
			}
			pr := i.PullRequest
			if pr != nil && pr.Number != nil {
				n := *pr.Number
				log.Infof("Creating fake PullRequest number %d at URL %s\n", n, pr.URL)
				prMap[n] = &gits.FakePullRequest{
					PullRequest: pr,
					Commits: []*gits.FakeCommit{
						{
							Commit: &gits.GitCommit{
								SHA:     testMergeSha,
								Message: "dummy commit",
							},
							Status: gits.CommitStatusPending,
						},
					},
					Comment: "comment for PR",
				}
			} else {
				log.Warnf("Missing number for PR %s\n", pr.URL)
			}
		}
		return i, err
	}

	fakeRepo := &gits.FakeRepository{
		GitRepo: &gits.GitRepository{
			Name: testRepoName,
		},
		PullRequests: prMap,
		Commits: []*gits.FakeCommit{
			{
				Commit: &gits.GitCommit{
					SHA:     testMergeSha,
					Message: "dummy commit",
				},
				Status: gits.CommitStatusPending,
			},
		},
	}
	fakeGitProvider := &gits.FakeProvider{
		Repositories: map[string][]*gits.FakeRepository{
			testOrgName: {fakeRepo},
		},
	}
	o := &ControllerWorkflowOptions{
		NoWatch:          true,
		FakePullRequests: fakePrFn,
		FakeGitProvider:  fakeGitProvider,
	}

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://github.com/jstrachan/environment-jstrachan-test1-staging.git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://github.com/jstrachan/environment-jstrachan-test1-production.git")
	staging.Spec.Order = 100
	production.Spec.Order = 200

	myFlowName := "myflow"

	step1 := workflow.CreateWorkflowPromoteStep("staging", false)
	step2 := workflow.CreateWorkflowPromoteStep("production", false, step1)

	ConfigureTestOptionsWithResources(&o.CommonOptions,
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
	o.git = &gits.GitFake{}

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
				assertPromoteStep(t, &spec.Steps[0], "staging", false)
			}
			if len(spec.Steps) > 1 {
				assertPromoteStep(t, &spec.Steps[1], "production", false)
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

	// lets make sure we don't create a PR for production as we have not completed the staging PR yet
	err = o.Run()
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// still no PR merged so cannot create a PR for production
	o.checkPullRequests(jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	pr1 := prMap[1]
	if !assert.NotNil(t, pr1, "Should have a Pull Request for 1") {
		return
	}

	// no PR on prod until staging completed
	setPullRequestMerged(pr1)

	// validate the fake git provider concurs
	testGitInfo := &gits.GitRepositoryInfo{
		Organisation: testOrgName,
		Name:         testRepoName,
	}
	pr, err := fakeGitProvider.GetPullRequest(testOrgName, testGitInfo, *pr1.PullRequest.Number)
	assert.NoError(t, err, "Finding PullRequest 1")
	assert.True(t, pr.Merged != nil && *pr.Merged, "Fake PR 1 is merged")

	o.checkPullRequests(jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	pr1 = prMap[1]
	setPromoteComplete(pr1, fakeRepo)

	// validate the fake git provider concurs
	statuses, err := fakeGitProvider.ListCommitStatus(testOrgName, testRepoName, *pr.MergeCommitSHA)
	assert.NoError(t, err, "Finding PullRequest 1 commit status")
	if assert.True(t, len(statuses) > 0, "PullRequest 1 statuses are not empty") {
		lastStatus := statuses[len(statuses)-1]
		assert.Equal(t, "success", lastStatus.State, "Last commit status of PullRequest 1 at %s", pr.URL)
	}

	o.checkPullRequests(jxClient, ns)

	// now lets run again due to change to the activity to detect the staging is complete
	err = o.Run()
	assertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)

	assertHasPullRequestForEnv(t, activities, a.Name, "production")
	assertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	assertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	pr2 := prMap[2]
	assert.NotNil(t, pr2, "No PullRequest 2 created")

	setPullRequestMerged(pr2)
	setPromoteComplete(pr2, fakeRepo)

	o.checkPullRequests(jxClient, ns)
	err = o.Run()

	// TODO
	//assertAllPromoteStepsSuccessful(t, activities, a.Name)
}

func setPromoteComplete(pr *gits.FakePullRequest, repository *gits.FakeRepository) {
	l := len(pr.Commits)
	if l > 0 {
		pr.Commits[l-1].Status = gits.CommitSatusSuccess

		repository.Commits[len(repository.Commits)-1] = pr.Commits[l-1]
		log.Infof("PR %s has commit status success\n", pr.PullRequest.URL)
	}
}

func setPullRequestMerged(pr *gits.FakePullRequest) {
	sha := testMergeSha
	merged := true
	pr.PullRequest.MergeCommitSHA = &sha
	pr.PullRequest.Merged = &merged

	log.Infof("PR %s is now merged\n", pr.PullRequest.URL)
}

func setPullRequestClosed(pr *gits.FakePullRequest) {
	now := time.Now()
	pr.PullRequest.ClosedAt = &now

	log.Infof("PR %s is now closed\n", pr.PullRequest.URL)
}

func createFakePullRequest(prNumber int) *gits.FakePullRequest {
	return &gits.FakePullRequest{
		PullRequest: &gits.GitPullRequest{
			Number: &prNumber,
			URL:    createFakePRURL(prNumber),
			Owner:  testOrgName,
			Repo:   testRepoName,
		},
		Commits: []*gits.FakeCommit{
			{
				Commit: &gits.GitCommit{
					SHA:     testMergeSha,
					Message: "dummy commit",
				},
				Status: gits.CommitStatusPending,
			},
		},
		Comment: "comment for PR",
	}
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
	for _, step := range activity.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			assert.Equal(t, string(v1.ActivityStatusTypeSucceeded), string(promote.Status), "PipelineActivity %s status for Promote to Environment %s", activity.Name, promote.Environment)
		}
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
	version := "1.0.1"
	a.Spec.GitOwner = folder
	a.Spec.GitRepository = repo
	a.Spec.GitURL = "https://github.com/" + folder + "/" + repo + ".git"
	a.Spec.Version = version
	a.Spec.Workflow = workflow
	_, err = activities.Update(a)
	return a, err
}

var fakePrCounter = 0

func createFakePullRequests(env *v1.Environment, modifyRequirementsFn ModifyRequirementsFn, branchNameText string, title string, message string, pullRequestInfo *ReleasePullRequestInfo) (*ReleasePullRequestInfo, error) {
	if pullRequestInfo == nil {
		pullRequestInfo = &ReleasePullRequestInfo{}
	}

	if pullRequestInfo.PullRequest == nil {
		pullRequestInfo.PullRequest = &gits.GitPullRequest{}
	}
	pr := pullRequestInfo.PullRequest
	if pr.Number == nil {
		fakePrCounter++
		n := fakePrCounter
		pr.Number = &n
	}
	if pr.URL == "" {
		pr.URL = createFakePRURL(*pr.Number)
	}
	if pr.Owner == "" {
		pr.Owner = testOrgName
	}
	if pr.Repo == "" {
		pr.Repo = testRepoName
	}
	log.Infof("Creating fake Pull Request for env %s branch %s title %s message %s with number %d and URL %s\n", env.Name, branchNameText, title, message, *pr.Number, pr.URL)
	return pullRequestInfo, nil
}

func createFakePRURL(i int) string {
	return "https://github.com/" + testOrgName + "/" + testRepoName + "/pulls/" + strconv.Itoa(i)
}
