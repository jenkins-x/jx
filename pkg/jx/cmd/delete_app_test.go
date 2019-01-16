package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
)

func TestDeleteAppForGitOps(t *testing.T) {
	testEnv, err := prepareDevEnv(t, true)
	assert.NoError(t, err)

	o := &cmd.DeleteAppOptions{
		CommonOptions:    *testEnv.CommonOptions,
		FakePullRequests: testEnv.FakePullRequests,
		GitOps:           true,
		DevEnv:           testEnv.DevEnv,
	}
	o.Args = []string{"example-app"}
	err = o.Run()
	assert.NoError(t, err)
	_, err = testEnv.FakeGitProvider.GetPullRequest(testEnv.OrgName, testEnv.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
}
