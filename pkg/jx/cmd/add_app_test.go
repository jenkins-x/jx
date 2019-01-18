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
	testEnv, err := prepareAppTests(t, true)
	defer func() {
		err := cleanupAppPRTests(t, testEnv)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testEnv.CommonOptions,
		},
		Version:    "0.0.1",
		Alias:      "example-alias",
		Repo:       "http://chartmuseum.jenkins-x.io",
		GitOps:     true,
		DevEnv:     testEnv.DevEnv,
		HelmUpdate: true, // Flag default when run on CLI
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
type AppTestEnv struct {
	CommonOptions   *cmd.CommonOptions
	FakeGitProvider *gits.FakeProvider
	DevRepo         *gits.FakeRepository
	DevEnvRepo      *gits.FakeRepository
	OrgName         string
	DevEnvRepoInfo  *gits.GitRepository
	DevEnv          *v1.Environment
}

func cleanupAppPRTests(t *testing.T, testEnv *AppTestEnv) error {
	err := cmd.CleanupTestResources(testEnv.CommonOptions)
	if err != nil {
		return err
	}
	return nil
}

func prepareAppTests(t *testing.T, gitOps bool) (*AppTestEnv, error) {
	testOrgName := "myorg"
	testRepoName := "my-app"
	devEnvRepoName := fmt.Sprintf("environment-%s-%s-dev", testOrgName, testRepoName)
	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	devEnvRepo := gits.NewFakeRepository(testOrgName, devEnvRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, devEnvRepo)

	o := cmd.AddAppOptions{}

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

	err := cmd.CreateTestEnvironmentDir(&o.CommonOptions)
	if err != nil {
		return nil, err
	}
	return &AppTestEnv{
		CommonOptions:   &o.CommonOptions,
		FakeGitProvider: fakeGitProvider,
		DevRepo:         fakeRepo,
		DevEnvRepo:      devEnvRepo,
		OrgName:         testOrgName,
		DevEnv:          devEnv,
		DevEnvRepoInfo: &gits.GitRepository{
			Name: devEnvRepoName,
		},
	}, nil

}
