package cmd_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/jx/cmd"

	"github.com/jenkins-x/jx/pkg/gits"
)

func TestUpgradeAppsForGitOps(t *testing.T) {
	testEnv, err := prepareAppTests(t, true)
	defer func() {
		err := cleanupAppPRTests(t, testEnv)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	// Now let's merge the

	o := &cmd.UpgradeAppsOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testEnv.CommonOptions,
		},
		Version: "0.0.1",
		Alias:   "example-alias",
		Repo:    "http://chartmuseum.jenkins-x.io",
		GitOps:  true,
		DevEnv:  testEnv.DevEnv,
	}
	o.Args = []string{"example-app"}

	err = o.Run()
	assert.NoError(t, err)
	_, err = testEnv.FakeGitProvider.GetPullRequest(testEnv.OrgName, testEnv.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
}

// Contains all useful data from the test environment initialized by `prepareInitialPromotionEnv`
type UpgradeAppsTestEnv struct {
	FakePullRequests cmd.CreateEnvPullRequestFn
	CommonOptions    *cmd.CommonOptions
	FakeGitProvider  *gits.FakeProvider
	DevRepo          *gits.FakeRepository
	DevEnvRepo       *gits.FakeRepository
	OrgName          string
	DevEnvRepoInfo   *gits.GitRepository
	DevEnv           *v1.Environment
}
