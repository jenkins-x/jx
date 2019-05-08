package cmd_test

import (
	"fmt"
	"github.com/satori/go.uuid"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx/pkg/jx/cmd/cmd_test_helpers"

	"github.com/petergtz/pegomock"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
)

func TestDeleteAppForGitOps(t *testing.T) {
	t.Parallel()
	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()

	testOptions := cmd_test_helpers.CreateAppTestOptions(true, name, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	name, alias, _, err := testOptions.DirectlyAddAppToGitOps(name, nil, "jx-app")
	assert.NoError(t, err)

	commonOpts := *testOptions.CommonOptions

	o := &cmd.DeleteAppOptions{
		CommonOptions:        &commonOpts,
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
	devEnvDir := testOptions.GetFullDevEnvDir(envDir)
	branchName, err := o.Git().Branch(devEnvDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("delete-app-%s", name), branchName)
	// Validate the updated Requirements.yaml
	requirements, err := helm.LoadRequirementsFile(filepath.Join(devEnvDir, helm.RequirementsFileName))
	assert.NoError(t, err)
	assert.Len(t, requirements.Dependencies, 0)
}

func TestDeleteAppWithShortNameForGitOps(t *testing.T) {
	t.Parallel()
	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, name, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	name, alias, _, err := testOptions.DirectlyAddAppToGitOps(name, nil, "jx-app")
	assert.NoError(t, err)
	shortName := strings.TrimPrefix(name, "jx-app-")
	// We also need to add the app CRD to Kubernetes -

	commonOpts := *testOptions.CommonOptions

	o := &cmd.DeleteAppOptions{
		CommonOptions:        &commonOpts,
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		ConfigureGitCallback: testOptions.ConfigureGitFn,
		Alias:                alias,
	}
	o.Args = []string{shortName}

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
	devEnvDir := testOptions.GetFullDevEnvDir(envDir)
	branchName, err := o.Git().Branch(devEnvDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("delete-app-%s", name), branchName)
	// Validate the updated Requirements.yaml
	requirements, err := helm.LoadRequirementsFile(filepath.Join(devEnvDir, helm.RequirementsFileName))
	assert.NoError(t, err)
	assert.Len(t, requirements.Dependencies, 0)
}

func TestDeleteApp(t *testing.T) {

	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)
	name, _, _, err := testOptions.AddApp(make(map[string]interface{}), "")
	assert.NoError(t, err)
	pegomock.RegisterMockTestingT(t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	commonOpts := *testOptions.CommonOptions
	o := &cmd.DeleteAppOptions{
		CommonOptions:        &commonOpts,
		DevEnv:               testOptions.DevEnv,
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}

	err = o.Run()
	assert.NoError(t, err)

	testOptions.MockHelmer.VerifyWasCalledOnce().
		DeleteRelease(pegomock.AnyString(), pegomock.EqString(fmt.Sprintf("%s-%s", namespace, name)), pegomock.AnyBool())
}

func TestDeleteAppWithShortName(t *testing.T) {

	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)
	name, _, _, err := testOptions.AddApp(make(map[string]interface{}), "")
	assert.NoError(t, err)
	shortName := strings.TrimPrefix(name, "jx-app-")
	pegomock.RegisterMockTestingT(t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	commonOpts := *testOptions.CommonOptions
	o := &cmd.DeleteAppOptions{
		CommonOptions:        &commonOpts,
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{shortName}

	err = o.Run()
	assert.NoError(t, err)

	testOptions.MockHelmer.VerifyWasCalledOnce().
		DeleteRelease(pegomock.AnyString(), pegomock.EqString(fmt.Sprintf("%s-%s", namespace, name)), pegomock.AnyBool())
}
