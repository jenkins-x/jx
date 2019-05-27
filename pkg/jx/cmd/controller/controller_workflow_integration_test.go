// +build integration

package controller_test

import (
	"encoding/json"
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/cmd_test_helpers"
	"github.com/jenkins-x/jx/pkg/jx/cmd/controller"
	"github.com/jenkins-x/jx/pkg/jx/cmd/promote"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/kube"
	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestSequentialWorkflow(t *testing.T) {
	originalJxHome, tempJxHome, err := cmd_test_helpers.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd_test_helpers.CleanupTestJxHomeDir(originalJxHome, tempJxHome)
		assert.NoError(t, err)
	}()
	originalKubeCfg, tempKubeCfg, err := cmd_test_helpers.CreateTestKubeConfigDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd_test_helpers.CleanupTestKubeConfigDir(originalKubeCfg, tempKubeCfg)
		assert.NoError(t, err)
	}()

	testOrgName := "jstrachan"
	testRepoName := "myrepo"
	stagingRepoName := "environment-staging"
	prodRepoName := "environment-production"

	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	stagingRepo := gits.NewFakeRepository(testOrgName, stagingRepoName)
	prodRepo := gits.NewFakeRepository(testOrgName, prodRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)
	fakeGitProvider.User.Username = testOrgName

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://fake.git/"+testOrgName+"/"+stagingRepoName+".git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://fake.git/"+testOrgName+"/"+prodRepoName+".git")
	staging.Spec.Order = 100
	production.Spec.Order = 200

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

	o := &controller.ControllerWorkflowOptions{
		CommonOptions:  &opts.CommonOptions{},
		NoWatch:        true,
		Namespace:      "jx",
		ConfigureGitFn: configureGitFn,
	}

	myFlowName := "myflow"

	step1 := workflow.CreateWorkflowPromoteStep("staging")
	step2 := workflow.CreateWorkflowPromoteStep("production", step1)

	cmd_test_helpers.ConfigureTestOptionsWithResources(o.CommonOptions,
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
		gits.NewGitLocal(),
		fakeGitProvider,
		helm.NewHelmCLI("helm", helm.V2, "", true),
		resources_test.NewMockInstaller(),
	)

	err = cmd_test_helpers.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)

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
				cmd_test_helpers.AssertPromoteStep(t, &spec.Steps[0], "staging")
			}
			if len(spec.Steps) > 1 {
				cmd_test_helpers.AssertPromoteStep(t, &spec.Steps[1], "production")
			}
		}
	}

	a, err := cmd_test_helpers.CreateTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", myFlowName)
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
	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, "staging")
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets make sure we don't create a PR for production as we have not completed the staging PR yet
	err = o.Run()
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// still no PR merged so cannot create a PR for production
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// test no PR on production until staging completed
	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, stagingRepo.Owner, stagingRepo.GitRepo.Name, 1) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 1) {
		return
	}

	// now lets poll again due to change to the activity to detect the staging is complete
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, "production")
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	cmd_test_helpers.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, prodRepo.Owner, prodRepo.GitRepo.Name, 1) {
		return
	}
	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, prodRepo, 1) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeSucceeded)

	cmd_test_helpers.AssertAllPromoteStepsSuccessful(t, activities, a.Name)
}

func TestWorkflowManualPromote(t *testing.T) {
	originalJxHome, tempJxHome, err := cmd_test_helpers.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd_test_helpers.CleanupTestJxHomeDir(originalJxHome, tempJxHome)
		assert.NoError(t, err)
	}()
	originalKubeCfg, tempKubeCfg, err := cmd_test_helpers.CreateTestKubeConfigDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd_test_helpers.CleanupTestKubeConfigDir(originalKubeCfg, tempKubeCfg)
		assert.NoError(t, err)
	}()

	testOrgName := "jstrachan"
	testRepoName := "manual"
	stagingRepoName := "environment-staging"
	prodRepoName := "environment-production"

	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	stagingRepo := gits.NewFakeRepository(testOrgName, stagingRepoName)
	prodRepo := gits.NewFakeRepository(testOrgName, prodRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)
	fakeGitProvider.User.Username = testOrgName

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://fake.git/"+testOrgName+"/"+stagingRepoName+".git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://fake.git/"+testOrgName+"/"+prodRepoName+".git")
	production.Spec.PromotionStrategy = v1.PromotionStrategyTypeManual

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

	o := &controller.ControllerWorkflowOptions{
		CommonOptions:  &opts.CommonOptions{},
		NoWatch:        true,
		Namespace:      "jx",
		ConfigureGitFn: configureGitFn,
	}

	workflowName := "default"

	cmd_test_helpers.ConfigureTestOptionsWithResources(o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			staging,
			production,
			kube.NewPreviewEnvironment("jx-jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment("jx-jstrachan-another-pr-3"),
		},
		gits.NewGitLocal(),
		fakeGitProvider,
		helm.NewHelmCLI("helm", helm.V2, "", true),
		resources_test.NewMockInstaller(),
	)

	err = cmd_test_helpers.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)

	a, err := cmd_test_helpers.CreateTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", workflowName)
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
	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, "staging")
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets make sure we don't create a PR for production as its manual
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, stagingRepo.Owner, stagingRepo.GitRepo.Name, 1) {
		return
	}
	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 1) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertAllPromoteStepsSuccessful(t, activities, a.Name)

	// now lets do a manual promotion
	version := a.Spec.Version
	po := &promote.PromoteOptions{
		Application:          testRepoName,
		Environment:          "production",
		Pipeline:             a.Spec.Pipeline,
		Build:                a.Spec.Build,
		Version:              version,
		NoPoll:               true,
		IgnoreLocalFiles:     true,
		HelmRepositoryURL:    helm.InClusterHelmRepositoryURL,
		LocalHelmRepoName:    kube.LocalHelmRepoName,
		Namespace:            "jx",
		ConfigureGitCallback: configureGitFn,
	}
	po.CommonOptions = o.CommonOptions
	po.BatchMode = true
	log.Infof("Promoting to production version %s for app %s\n", version, testRepoName)
	err = po.Run()
	assert.NoError(t, err)
	if err != nil {
		return
	}

	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, "production")
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)
	cmd_test_helpers.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, prodRepo.Owner, prodRepo.GitRepo.Name, 1) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeSucceeded)

	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, prodRepo, 1) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertAllPromoteStepsSuccessful(t, activities, a.Name)
}

// TestParallelWorkflow lets test promoting to A + B then when A + B is complete then C
func TestParallelWorkflow(t *testing.T) {
	originalJxHome, tempJxHome, err := cmd_test_helpers.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd_test_helpers.CleanupTestJxHomeDir(originalJxHome, tempJxHome)
		assert.NoError(t, err)
	}()
	originalKubeCfg, tempKubeCfg, err := cmd_test_helpers.CreateTestKubeConfigDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd_test_helpers.CleanupTestKubeConfigDir(originalKubeCfg, tempKubeCfg)
		assert.NoError(t, err)
	}()

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

	envA := kube.NewPermanentEnvironmentWithGit(envNameA, "https://fake.git/"+testOrgName+"/"+envRepoNameA+".git")
	envB := kube.NewPermanentEnvironmentWithGit(envNameB, "https://fake.git/"+testOrgName+"/"+envRepoNameB+".git")
	envC := kube.NewPermanentEnvironmentWithGit(envNameC, "https://fake.git/"+testOrgName+"/"+envRepoNameC+".git")

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
		data, err := json.Marshal(envA)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, "templates", fmt.Sprintf("%s.yaml", envRepoNameA)), data, 0755)
		if err != nil {
			return err
		}
		data, err = json.Marshal(envB)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, "templates", fmt.Sprintf("%s.yaml", envRepoNameB)), data, 0755)
		if err != nil {
			return err
		}
		data, err = json.Marshal(envC)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, "templates", fmt.Sprintf("%s.yaml", envRepoNameC)), data, 0755)
		if err != nil {
			return err
		}
		return gitter.AddCommit(dir, "Initial Commit")
	}

	o := &controller.ControllerWorkflowOptions{
		CommonOptions:  &opts.CommonOptions{},
		NoWatch:        true,
		Namespace:      "jx",
		ConfigureGitFn: configureGitFn,
	}

	myFlowName := "myflow"

	step1 := workflow.CreateWorkflowPromoteStep(envNameA)
	step2 := workflow.CreateWorkflowPromoteStep(envNameB)
	step3 := workflow.CreateWorkflowPromoteStep(envNameC, step1, step2)

	cmd_test_helpers.ConfigureTestOptionsWithResources(o.CommonOptions,
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
		gits.NewGitLocal(),
		fakeGitProvider,
		helm.NewHelmCLI("helm", helm.V2, "", true),
		resources_test.NewMockInstaller(),
	)
	err = cmd_test_helpers.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)

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
				cmd_test_helpers.AssertPromoteStep(t, &spec.Steps[0], envNameA)
			}
			if len(spec.Steps) > 1 {
				cmd_test_helpers.AssertPromoteStep(t, &spec.Steps[1], envNameB)
			}
			if len(spec.Steps) > 2 {
				cmd_test_helpers.AssertPromoteStep(t, &spec.Steps[2], envNameC)
			}
		}
	}

	a, err := cmd_test_helpers.CreateTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", myFlowName)
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
	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, envNameA)
	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, envNameB)
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets make sure we don't create a PR for production as we have not completed the staging PR yet
	err = o.Run()
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, envNameC)

	// still no PR merged so cannot create a PR for C until A and B complete
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, envNameC)

	// test no PR on production until staging completed
	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, repoA.Owner, repoA.GitRepo.Name, 1) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, envNameC)

	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, repoA, 1) {
		return
	}

	// now lets poll again due to change to the activity to detect the staging is complete
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, envNameC)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, envNameA, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, envNameB, v1.ActivityStatusTypeRunning)
	cmd_test_helpers.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, repoB.Owner, repoB.GitRepo.Name, 1) {
		return
	}
	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, repoB, 1) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	// C should have started now
	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, envNameC)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, envNameA, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, envNameB, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, envNameC, v1.ActivityStatusTypeRunning)

	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, repoC.Owner, repoC.GitRepo.Name, 1) {
		return
	}
	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, repoC, 1) {
		return
	}

	// should be complete now
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, envNameA, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, envNameB, v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, envNameC, v1.ActivityStatusTypeSucceeded)

	cmd_test_helpers.AssertAllPromoteStepsSuccessful(t, activities, a.Name)
}

// TestNewVersionWhileExistingWorkflow lets test that we create a new workflow and terminate
// the old workflow if we find a new version
func TestNewVersionWhileExistingWorkflow(t *testing.T) {
	originalJxHome, tempJxHome, err := cmd_test_helpers.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd_test_helpers.CleanupTestJxHomeDir(originalJxHome, tempJxHome)
		assert.NoError(t, err)
	}()
	originalKubeCfg, tempKubeCfg, err := cmd_test_helpers.CreateTestKubeConfigDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd_test_helpers.CleanupTestKubeConfigDir(originalKubeCfg, tempKubeCfg)
		assert.NoError(t, err)
	}()

	testOrgName := "jstrachan"
	testRepoName := "myrepo"
	stagingRepoName := "environment-staging"
	prodRepoName := "environment-production"

	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	stagingRepo := gits.NewFakeRepository(testOrgName, stagingRepoName)
	prodRepo := gits.NewFakeRepository(testOrgName, prodRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://fake.git/"+testOrgName+"/"+stagingRepoName+".git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://fake.git/"+testOrgName+"/"+prodRepoName+".git")
	staging.Spec.Order = 100
	production.Spec.Order = 200

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

	o := &controller.ControllerWorkflowOptions{
		CommonOptions:  &opts.CommonOptions{},
		NoWatch:        true,
		Namespace:      "jx",
		ConfigureGitFn: configureGitFn,
	}

	myFlowName := "myflow"

	step1 := workflow.CreateWorkflowPromoteStep("staging")
	step2 := workflow.CreateWorkflowPromoteStep("production", step1)

	cmd_test_helpers.ConfigureTestOptionsWithResources(o.CommonOptions,
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
		gits.NewGitLocal(),
		fakeGitProvider,
		helm.NewHelmCLI("helm", helm.V2, "", true),
		resources_test.NewMockInstaller(),
	)
	err = cmd_test_helpers.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)

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
				cmd_test_helpers.AssertPromoteStep(t, &spec.Steps[0], "staging")
			}
			if len(spec.Steps) > 1 {
				cmd_test_helpers.AssertPromoteStep(t, &spec.Steps[1], "production")
			}
		}
	}

	a, err := cmd_test_helpers.CreateTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "1", myFlowName)
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
	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, "staging")
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets trigger a new pipeline release which should close the old version
	aOld := a
	a, err = cmd_test_helpers.CreateTestPipelineActivity(jxClient, ns, testOrgName, testRepoName, "master", "2", myFlowName)

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, "staging")
	cmd_test_helpers.AssertWorkflowStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	// lets make sure we don't create a PR for production as we have not completed the staging PR yet
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	cmd_test_helpers.AssertWorkflowStatus(t, activities, aOld.Name, v1.ActivityStatusTypeAborted)

	// still no PR merged so cannot create a PR for production
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	// test no PR on production until staging completed
	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, stagingRepo.Owner, stagingRepo.GitRepo.Name, 2) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)
	cmd_test_helpers.AssertHasNoPullRequestForEnv(t, activities, a.Name, "production")

	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, stagingRepo, 2) {
		return
	}

	// now lets poll again due to change to the activity to detect the staging is complete
	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPullRequestForEnv(t, activities, a.Name, "production")
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeRunning)
	cmd_test_helpers.AssertHasPipelineStatus(t, activities, a.Name, v1.ActivityStatusTypeRunning)

	if !cmd_test_helpers.AssertSetPullRequestMerged(t, fakeGitProvider, prodRepo.Owner, prodRepo.GitRepo.Name, 1) {
		return
	}
	if !cmd_test_helpers.AssertSetPullRequestComplete(t, fakeGitProvider, prodRepo, 1) {
		return
	}

	cmd_test_helpers.PollGitStatusAndReactToPipelineChanges(t, o, jxClient, ns)

	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "staging", v1.ActivityStatusTypeSucceeded)
	cmd_test_helpers.AssertHasPromoteStatus(t, activities, a.Name, "production", v1.ActivityStatusTypeSucceeded)

	cmd_test_helpers.AssertAllPromoteStepsSuccessful(t, activities, a.Name)
}

func TestPullRequestNumber(t *testing.T) {
	failUrls := []string{"https://fake.git/foo/bar/pulls"}
	for _, u := range failUrls {
		_, err := controller.PullRequestURLToNumber(u)
		assert.Errorf(t, err, "Expected error for pullRequestURLToNumber() with %s", u)
	}

	tests := map[string]int{
		"https://fake.git/foo/bar/pulls/12": 12,
	}

	for u, expected := range tests {
		actual, err := controller.PullRequestURLToNumber(u)
		assert.NoError(t, err, "pullRequestURLToNumber() should not fail for %s", u)
		if err == nil {
			assert.Equal(t, expected, actual, "pullRequestURLToNumber() for %s", u)
		}
	}
}
