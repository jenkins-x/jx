package cmd_test

import (
	"fmt"
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jenkins-x/jx/pkg/gits"
)

func TestAddAppForGitOps(t *testing.T) {
	testEnv, err := prepareDevEnv(t, true)
	assert.NoError(t, err)

	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testEnv.CommonOptions,
		},
		FakePullRequests: testEnv.FakePullRequests,
		Version:          "0.0.1",
		Alias:            "example-alias",
		Repo:             "http://chartmuseum.jenkins-x.io",
		GitOps:           true,
		DevEnv:           testEnv.DevEnv,
		HelmUpdate:       true, // Flag default when run on CLI
	}
	o.Args = []string{"example-app"}
	// Set by flag defaults
	o.HelmUpdate = true
	err = o.Run()
	assert.NoError(t, err)
	_, err = testEnv.FakeGitProvider.GetPullRequest(testEnv.OrgName, testEnv.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
}

// Contains all useful data from the test environment initialized by `prepareInitialPromotionEnv`
type AddAppTestEnv struct {
	FakePullRequests cmd.CreateEnvPullRequestFn
	CommonOptions    *cmd.CommonOptions
	FakeGitProvider  *gits.FakeProvider
	DevRepo          *gits.FakeRepository
	DevEnvRepo       *gits.FakeRepository
	OrgName          string
	DevEnvRepoInfo   *gits.GitRepository
	DevEnv           *v1.Environment
}

func prepareDevEnv(t *testing.T, gitOps bool) (*AddAppTestEnv, error) {
	testOrgName := "myorg"
	testRepoName := "my-app"
	devEnvRepoName := fmt.Sprintf("environment-%s-%s-dev", testOrgName, testRepoName)
	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	devEnvRepo := gits.NewFakeRepository(testOrgName, devEnvRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, devEnvRepo)

	o := cmd.AddAppOptions{
		FakePullRequests: cmd.NewCreateEnvPullRequestFn(fakeGitProvider),
	}

	devEnv := kube.NewPermanentEnvironmentWithGit("dev", fmt.Sprintf("https://github.com/%s/%s.git", testOrgName,
		devEnvRepoName))
	if gitOps {
		devEnv.Spec.Source.URL = devEnvRepo.GitRepo.CloneURL
		devEnv.Spec.Source.Ref = "master"
	}

	cmd.ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			devEnv,
		},
		&gits.GitFake{},
		helm_test.NewMockHelmer(),
	)
	return &AddAppTestEnv{
		FakePullRequests: o.FakePullRequests,
		CommonOptions:    &o.CommonOptions,
		FakeGitProvider:  fakeGitProvider,
		DevRepo:          fakeRepo,
		DevEnvRepo:       devEnvRepo,
		OrgName:          testOrgName,
		DevEnv:           devEnv,
		DevEnvRepoInfo: &gits.GitRepository{
			Name: devEnvRepoName,
		},
	}, nil

}
