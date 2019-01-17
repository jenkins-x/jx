package cmd_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/petergtz/pegomock"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
)

func TestDeleteAppForGitOps(t *testing.T) {
	t.Parallel()
	testOptions := CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	name, alias, _, err := testOptions.AddApp()
	assert.NoError(t, err)

	o := &cmd.DeleteAppOptions{
		CommonOptions:        *testOptions.CommonOptions,
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		ConfigureGitCallback: testOptions.ConfigureGitFn,
		Alias:                alias,
	}
	o.Args = []string{name}

	err = o.Run()
	assert.NoError(t, err)
	// Validate a PR was created
	pr, err := testOptions.FakeGitProvider.GetPullRequest(testOptions.OrgName, testOptions.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
	// Validate the PR has the right title, message
	assert.Equal(t, fmt.Sprintf("Delete %s", name), pr.Title)
	assert.Equal(t, fmt.Sprintf("Delete app %s", name), pr.Body)
	// Validate the branch name
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
	branchName, err := o.Git().Branch(devEnvDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("delete-app-%s", name), branchName)
	// Validate the updated Requirements.yaml
	requirements, err := helm.LoadRequirementsFile(filepath.Join(devEnvDir, helm.RequirementsFileName))
	assert.NoError(t, err)
	assert.Len(t, requirements.Dependencies, 1)
	assert.Nil(t, requirements.Dependencies[0])
}

func TestDeleteApp(t *testing.T) {

	testOptions := CreateAppTestOptions(false, t)
	// Can't run in parallel
	pegomock.RegisterMockTestingT(t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name, _, _, err := testOptions.AddApp()
	assert.NoError(t, err)

	o := &cmd.DeleteAppOptions{
		CommonOptions:        *testOptions.CommonOptions,
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}

	err = o.Run()
	assert.NoError(t, err)

	testOptions.MockHelmer.VerifyWasCalledOnce().
		DeleteRelease(pegomock.AnyString(), pegomock.EqString(name), pegomock.AnyBool())
}
