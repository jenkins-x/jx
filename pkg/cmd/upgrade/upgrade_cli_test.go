package upgrade_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx/pkg/cmd/upgrade"
	"github.com/jenkins-x/jx/pkg/cmd/version"

	"github.com/blang/semver"

	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	// typev1 "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/typed/jenkins.io/v1"
)

func TestNeedsUpgrade(t *testing.T) {
	type testData struct {
		current               string
		latest                string
		expectedUpgradeNeeded bool
		expectedMessage       string
	}

	testCases := []testData{
		{
			"1.0.0", "1.0.0", false, "You are already on the latest version of jx 1.0.0\n",
		},
		{
			"1.0.0", "1.0.1", true, "",
		},
		{
			"1.0.0", "0.0.99", true, "",
		},
	}

	o := upgrade.CLIOptions{}
	for _, data := range testCases {
		currentVersion, _ := semver.New(data.current)
		latestVersion, _ := semver.New(data.latest)
		actualMessage := log.CaptureOutput(func() {
			actualUpgradeNeeded := o.NeedsUpgrade(*currentVersion, *latestVersion)
			assert.Equal(t, data.expectedUpgradeNeeded, actualUpgradeNeeded, fmt.Sprintf("Unexpected upgrade flag for %v", data))
		})
		assert.Equal(t, data.expectedMessage, actualMessage, fmt.Sprintf("Unexpected message for %v", data))
	}
}

func TestNewCmdUpgradeCLI(t *testing.T) {
	kubeClient := jxfake.NewSimpleClientset()
	os.Setenv("KUBECONFIG", "testdata/kubeconfig")
	cmd, o := upgrade.NewCmdUpgradeCLI()
	o.JXClient = kubeClient
	o.Version = "3.2.238"
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestVersionCheckWhenCurrentVersionIsGreaterThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	version.Version = "1.4.0"
	opts := &upgrade.CLIOptions{}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersionCheckWhenCurrentVersionIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Version = "1.2.3"
	opts := &upgrade.CLIOptions{}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersionCheckWhenCurrentVersionIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	version.Version = "1.0.0"
	opts := &upgrade.CLIOptions{}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.True(t, update, "should update")
}

func TestVersionCheckWhenCurrentVersionIsEqualToReleaseVersionWithPatch(t *testing.T) {
	var prVersions []semver.PRVersion
	prVersions = append(prVersions, semver.PRVersion{VersionStr: "dev"})
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3, Pre: prVersions, Build: []string(nil)}
	version.Version = "1.2.3"
	opts := &upgrade.CLIOptions{}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersionCheckWhenCurrentVersionWithPatchIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Version = "1.2.3-dev+6a8285f4"
	opts := &upgrade.CLIOptions{}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersionCheckWhenCurrentVersionWithPatchIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Version = "1.2.2-dev+6a8285f4"
	opts := &upgrade.CLIOptions{}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}
