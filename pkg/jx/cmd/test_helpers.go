package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	v1fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apifake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/ghodss/yaml"
)

// ConfigureTestOptions lets configure the options for use in tests
// using fake APIs to k8s cluster
func ConfigureTestOptions(o *CommonOptions, git gits.Gitter, helm helm.Helmer) {
	ConfigureTestOptionsWithResources(o, nil, nil, git, helm)
}

// ConfigureTestOptions lets configure the options for use in tests
// using fake APIs to k8s cluster
func ConfigureTestOptionsWithResources(o *CommonOptions, k8sObjects []runtime.Object,
	jxObjects []runtime.Object, git gits.Gitter, helm helm.Helmer) {
	//o.Out = tests.Output()
	o.BatchMode = true
	if o.Factory == nil {
		o.Factory = NewFactory()
	}
	o.currentNamespace = "jx"

	namespacesRequired := []string{o.currentNamespace}
	namespaceMap := map[string]*corev1.Namespace{}

	for _, ro := range k8sObjects {
		ns, ok := ro.(*corev1.Namespace)
		if ok {
			namespaceMap[ns.Name] = ns
		}
	}
	hasDev := false
	for _, ro := range jxObjects {
		env, ok := ro.(*v1.Environment)
		if ok {
			ns := env.Spec.Namespace
			if ns != "" && util.StringArrayIndex(namespacesRequired, ns) < 0 {
				namespacesRequired = append(namespacesRequired, ns)
			}
			if env.Name == "dev" {
				hasDev = true
			}
		}
	}

	// ensure we've the dev nenvironment
	if !hasDev {
		devEnv := kube.NewPermanentEnvironment("dev")
		devEnv.Spec.Namespace = o.currentNamespace
		devEnv.Spec.Kind = v1.EnvironmentKindTypeDevelopment

		jxObjects = append(jxObjects, devEnv)
	}

	// add any missing namespaces
	for _, ns := range namespacesRequired {
		if namespaceMap[ns] == nil {
			k8sObjects = append(k8sObjects, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
					Labels: map[string]string{
						"tag": "",
					},
				},
			})
		}
	}

	client := fake.NewSimpleClientset(k8sObjects...)
	o.SetKubeClient(client)
	o.jxClient = v1fake.NewSimpleClientset(jxObjects...)
	o.apiExtensionsClient = apifake.NewSimpleClientset()
	o.git = git
	o.helm = helm
}

func NewCreateEnvPullRequestFn(provider *gits.FakeProvider) CreateEnvPullRequestFn {
	fakePrFn := func(env *v1.Environment, modifyChartFn ModifyChartFn, branchNameText string, title string, message string,
		pullRequestInfo *gits.PullRequestInfo) (*gits.PullRequestInfo, error) {
		envURL := env.Spec.Source.URL
		values := []string{}
		for _, repos := range provider.Repositories {
			for _, repo := range repos {
				cloneURL := repo.GitRepo.CloneURL
				if cloneURL == envURL {
					return createFakePullRequest(repo, env, modifyChartFn, branchNameText, title, message, pullRequestInfo, provider)
				}
				values = append(values, cloneURL)
			}
		}
		return nil, fmt.Errorf("Could not find repository for cloneURL %s values found %s", envURL, strings.Join(values, ", "))
	}
	return fakePrFn
}

func CreateTestPipelineActivity(jxClient versioned.Interface, ns string, folder string, repo string, branch string, build string, workflow string) (*v1.PipelineActivity, error) {
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

func createFakePullRequest(repository *gits.FakeRepository, env *v1.Environment, modifyChartFn ModifyChartFn,
	branchNameText string, title string, message string, pullRequestInfo *gits.PullRequestInfo, provider *gits.FakeProvider) (*gits.PullRequestInfo, error) {
	if pullRequestInfo == nil {
		pullRequestInfo = &gits.PullRequestInfo{}
	}

	if pullRequestInfo.GitProvider == nil {
		pullRequestInfo.GitProvider = provider
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

func AssertHasPullRequestForEnv(t *testing.T, activities typev1.PipelineActivityInterface, name string, envName string) {
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

func WaitForPullRequestForEnv(t *testing.T, activities typev1.PipelineActivityInterface, name string, envName string) {
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		assert.NoError(t, err, "Could not find PipelineActivity %s", name)
		return
	}
	waitTime, _ := time.ParseDuration("20s")
	end := time.Now().Add(waitTime)
	for {
		for _, step := range activity.Spec.Steps {
			promote := step.Promote
			if promote != nil {
				if promote.Environment == envName {
					failed := false
					pullRequestStep := promote.PullRequest
					if pullRequestStep == nil {
						failed = true
					}
					u := pullRequestStep.PullRequestURL
					log.Infof("Found Promote PullRequest %s on PipelineActivity %s for Environment %s\n", u, name, envName)

					if !assert.True(t, u != "", "No PullRequest URL on PipelineActivity %s for Promote step for Environment %s", name, envName) {
						failed = true
					}
					if !failed {
						return
					}

				}
			}
		}
		if time.Now().After(end) {
			log.Infof("No Promote PR found on PipelineActivity %s for Environment %s\n", name, envName)
			//assert.Fail(t, "Missing Promote PR", "No Promote PR found on PipelineActivity %s for Environment %s", name, envName)
			//dumpFailedActivity(activity)
			return
		}
		log.Infof("Waiting 1s for PullRequest in Enviroment %s\n", envName)
		v, _ := time.ParseDuration("2s")
		time.Sleep(v)
		activity, _ = activities.Get(name, metav1.GetOptions{})
	}
}

func AssertWorkflowStatus(t *testing.T, activities typev1.PipelineActivityInterface, name string, status v1.ActivityStatusType) {
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

func AssertSetPullRequestComplete(t *testing.T, provider *gits.FakeProvider, repository *gits.FakeRepository, prNumber int) bool {
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

	// validate the fake Git provider concurs
	repoOwner := repository.Owner
	repoName := repository.Name()
	testGitInfo := &gits.GitRepository{
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

func SetSuccessCommitStatusInPR(t *testing.T, repository *gits.FakeRepository, prNumber int) {
	fakePR := repository.PullRequests[prNumber]
	assert.NotNil(t, fakePR, "No PullRequest found on repository %s for number #%d", repository.String(), prNumber)

	l := len(fakePR.Commits)
	if l > 0 {
		fakePR.Commits[l-1].Status = gits.CommitSatusSuccess
	}
}

func AssertHasPromoteStatus(t *testing.T, activities typev1.PipelineActivityInterface, name string, envName string, status v1.ActivityStatusType) {
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

func AssertHasPipelineStatus(t *testing.T, activities typev1.PipelineActivityInterface, name string, status v1.ActivityStatusType) {
	activity, err := activities.Get(name, metav1.GetOptions{})
	if err != nil {
		assert.NoError(t, err, "Could not find PipelineActivity %s", name)
		return
	}
	if !assert.Equal(t, string(status), string(activity.Spec.Status), "activity status for PipelineActivity %s", name) {
		dumpFailedActivity(activity)
	}
}

func AssertAllPromoteStepsSuccessful(t *testing.T, activities typev1.PipelineActivityInterface, name string) {
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

func AssertHasNoPullRequestForEnv(t *testing.T, activities typev1.PipelineActivityInterface, name string, envName string) {
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

func SetPullRequestClosed(pr *gits.FakePullRequest) {
	now := time.Now()
	pr.PullRequest.ClosedAt = &now

	log.Infof("PR %s is now closed\n", pr.PullRequest.URL)
}

func AssertSetPullRequestMerged(t *testing.T, provider *gits.FakeProvider, repository *gits.FakeRepository, prNumber int) bool {
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

	// validate the fake Git provider concurs
	testGitInfo := &gits.GitRepository{
		Organisation: repository.Owner,
		Name:         repository.Name(),
	}
	pr, err := provider.GetPullRequest(repository.Owner, testGitInfo, prNumber)
	assert.NoError(t, err, "Finding PullRequest %d", prNumber)
	return assert.True(t, pr.Merged != nil && *pr.Merged, "Fake PR %d is merged", prNumber)
}

func AssertPromoteStep(t *testing.T, step *v1.WorkflowStep, expectedEnvironment string) {
	promote := step.Promote
	assert.True(t, promote != nil, "step is a promote step")

	if promote != nil {
		assert.Equal(t, expectedEnvironment, promote.Environment, "environment name")
	}
}

func PollGitStatusAndReactToPipelineChanges(t *testing.T, o *ControllerWorkflowOptions, jxClient versioned.Interface, ns string) error {
	o.ReloadAndPollGitPipelineStatuses(jxClient, ns)
	err := o.Run()
	assert.NoError(t, err, "Failed to react to PipelineActivity changes")
	return err
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

func dumpFailedActivity(activity *v1.PipelineActivity) {
	data, err := yaml.Marshal(activity)
	if err == nil {
		log.Warnf("YAML: %s\n", string(data))
	}
}

func dumpPipelineMap(o *ControllerWorkflowOptions) {
	log.Infof("Dumping PipelineMap {\n")
	for k, v := range o.PipelineMap() {
		log.Infof("    Pipeline %s %s\n", k, v.Name)
	}
	log.Infof("}\n")
}
