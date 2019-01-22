package cmd_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/blang/semver"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
)

func TestUpgradeAppForGitOps(t *testing.T) {
	t.Parallel()
	testOptions, err := cmd.CreateAppTestOptions(true)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	name, alias, version, err := testOptions.AddApp()
	assert.NoError(t, err)

	// Now let's upgrade

	newVersion, err := semver.Parse(version)
	assert.NoError(t, err)
	newVersion.Patch++
	o := &cmd.UpgradeAppsOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
		},
		Version:              newVersion.String(),
		Alias:                alias,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}

	err = o.Run()
	assert.NoError(t, err)
	// Validate a PR was created
	pr, err := testOptions.FakeGitProvider.GetPullRequest(testOptions.OrgName, testOptions.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
	// Validate the PR has the right title, message
	assert.Equal(t, fmt.Sprintf("Upgrade %s to %s", name, newVersion.String()), pr.Title)
	assert.Equal(t, fmt.Sprintf("Upgrade %s from %s to %s", name, version, newVersion.String()), pr.Body)
	// Validate the branch name
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
	branchName, err := o.Git().Branch(devEnvDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("upgrade-app-%s-%s", name, newVersion.String()), branchName)
	// Validate the updated Requirements.yaml
	requirements, err := helm.LoadRequirementsFile(filepath.Join(devEnvDir, helm.RequirementsFileName))
	assert.NoError(t, err)
	found := make([]*helm.Dependency, 0)
	for _, d := range requirements.Dependencies {
		if d.Name == name && d.Alias == alias {
			found = append(found, d)
		}
	}
	assert.Len(t, found, 1)
	assert.Equal(t, newVersion.String(), found[0].Version)
}

func TestUpgradeAppToLatestForGitOps(t *testing.T) {
	testOptions, err := cmd.CreateAppTestOptions(true)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	name, alias, version, err := testOptions.AddApp()
	assert.NoError(t, err)

	// Now let's upgrade

	newVersion, err := semver.Parse(version)
	assert.NoError(t, err)
	newVersion.Patch++
	o := &cmd.UpgradeAppsOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
		},
		Version:              newVersion.String(),
		Alias:                alias,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}

	helm_test.StubFetchChart(name, "", cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: newVersion.String(),
		},
	}, testOptions.MockHelmer)

	err = o.Run()
	assert.NoError(t, err)
	// Validate a PR was created
	pr, err := testOptions.FakeGitProvider.GetPullRequest(testOptions.OrgName, testOptions.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
	// Validate the PR has the right title, message
	assert.Equal(t, fmt.Sprintf("Upgrade %s to %s", name, newVersion.String()), pr.Title)
	assert.Equal(t, fmt.Sprintf("Upgrade %s from %s to %s", name, version, newVersion.String()), pr.Body)
	// Validate the branch name
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
	branchName, err := o.Git().Branch(devEnvDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("upgrade-app-%s-%s", name, newVersion.String()), branchName)
	// Validate the updated Requirements.yaml
	requirements, err := helm.LoadRequirementsFile(filepath.Join(devEnvDir, helm.RequirementsFileName))
	assert.NoError(t, err)
	found := make([]*helm.Dependency, 0)
	for _, d := range requirements.Dependencies {
		if d.Name == name && d.Alias == alias {
			found = append(found, d)
		}
	}
	assert.Len(t, found, 1)
	assert.Equal(t, newVersion.String(), found[0].Version)
}

func TestUpgradeAllAppsForGitOps(t *testing.T) {
	testOptions, err := cmd.CreateAppTestOptions(true)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	name1, alias1, version1, err := testOptions.AddApp()
	assert.NoError(t, err)
	name2, alias2, version2, err := testOptions.AddApp()
	assert.NoError(t, err)

	newVersion1, err := semver.Parse(version1)
	assert.NoError(t, err)
	newVersion1.Patch++

	newVersion2, err := semver.Parse(version2)
	assert.NoError(t, err)
	newVersion2.Minor++

	// Now let's upgrade
	o := &cmd.UpgradeAppsOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
		},
		Repo:                 cmd.DEFAULT_CHARTMUSEUM_URL,
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}

	helm_test.StubFetchChart(name1, "", cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name1,
			Version: newVersion1.String(),
		},
	}, testOptions.MockHelmer)

	helm_test.StubFetchChart(name2, "",
		cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
			Metadata: &chart.Metadata{
				Name:    name2,
				Version: newVersion2.String(),
			},
		}, testOptions.MockHelmer)

	err = o.Run()
	assert.NoError(t, err)
	// Validate a PR was created
	pr, err := testOptions.FakeGitProvider.GetPullRequest(testOptions.OrgName, testOptions.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
	// Validate the PR has the right title, message
	assert.Equal(t, fmt.Sprintf("Upgrade all apps"), pr.Title)
	assert.Equal(t, fmt.Sprintf("Upgrade all apps:\n\n* %s from %s to %s\n* %s from %s to %s", name1,
		version1,
		newVersion1.String(), name2, version2, newVersion2.String()), pr.Body)
	// Validate the branch name1
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
	branchName, err := o.Git().Branch(devEnvDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("upgrade-all-apps"), branchName)
	// Validate the updated Requirements.yaml
	requirements, err := helm.LoadRequirementsFile(filepath.Join(devEnvDir, helm.RequirementsFileName))
	assert.NoError(t, err)
	found := make([]*helm.Dependency, 0)
	for _, d := range requirements.Dependencies {
		if d.Name == name1 && d.Alias == alias1 {
			found = append(found, d)
			assert.Equal(t, newVersion1.String(), d.Version)
		}
		if d.Name == name2 && d.Alias == alias2 {
			found = append(found, d)
			assert.Equal(t, newVersion2.String(), d.Version)
		}
	}
	assert.Len(t, found, 2)
}
