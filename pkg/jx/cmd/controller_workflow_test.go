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
	pr1 := createFakePullRequest(1)
	prMap := map[int]*gits.FakePullRequest{
		1: pr1,
	}
	fakeRepo := &gits.FakeRepository{
		GitRepo: &gits.GitRepository{
			Name: testRepoName,
		},
		PullRequests: prMap,
		Commits:      pr1.Commits,
	}
	fakeGitProvider := &gits.FakeProvider{
		Repositories: map[string][]*gits.FakeRepository{
			testOrgName: {fakeRepo},
		},
	}
	o := &ControllerWorkflowOptions{
		NoWatch:          true,
		FakePullRequests: createFakePullRequests,
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

	// no PR on prod until staging completed
	setPullRequestMerged(pr1)
	o.checkPullRequests(jxClient, ns)
	assertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	setPromoteComplete(pr1)
	o.checkPullRequests(jxClient, ns)

	// now lets run again due to change to the activity to detect the staging is complete
	err = o.Run()
	assertHasPullRequestForEnv(t, activities, a.Name, "production")
}

func setPromoteComplete(pr *gits.FakePullRequest) {
	l := len(pr.Commits)
	if l > 0 {
		pr.Commits[l-1].Status = gits.CommitSatusSuccess

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
	found := false
	for _, step := range activity.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			if promote.Environment == envName {
				failed := false
				pullRequestStep := promote.PullRequest
				if pullRequestStep == nil {
					assert.Fail(t, "No PullRequest object on Promote step for Environment %s", envName)
					failed = true
				}
				u := pullRequestStep.PullRequestURL
				log.Infof("Found Promote PullRequest %s for Environment %s\n", u, envName)

				if !assert.True(t, u != "", "No PullRequest URL on Promote step for Environment %s", envName) {
					failed = true
				}
				if failed {
					dumpFailedActivity(activity)
				}
				return
			}
		}
	}
	if !assert.True(t, found, "No Promote PullRequest found for Environment %s", envName) {
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

	log.Infof("Creating fake Pull Request for env %s branch %s title %s message %s\n", env.Name, branchNameText, title, message)

	if pullRequestInfo.PullRequest == nil {
		pullRequestInfo.PullRequest = &gits.GitPullRequest{}
	}
	pr := pullRequestInfo.PullRequest
	if pr.URL == "" {
		fakePrCounter++
		pr.URL = createFakePRURL(fakePrCounter)
	}
	return pullRequestInfo, nil
}

func createFakePRURL(i int) string {
	return "https://github.com/" + testOrgName + "/" + testRepoName + "/pulls/" + strconv.Itoa(i)
}
