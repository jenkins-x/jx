package cmd_test

import (
	"io"
	"reflect"
	"testing"

	"github.com/blang/semver"
	"github.com/bouk/monkey"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// TODO reafcator. setup function makes tests sequence dependent. Tests cannot be run with t.Parallel()
func setup(latestJXVersion semver.Version) {
	var o *commoncmd.CommonOptions
	monkey.PatchInstanceMethod(reflect.TypeOf(o), "GetLatestJXVersion", func(*commoncmd.CommonOptions) (semver.Version, error) {
		return latestJXVersion, nil
	})
	monkey.Patch(util.ColorInfo, func(input ...interface{}) string {
		return "ColourInfo"
	})
	monkey.Patch(util.Confirm, func(message string, b bool, m string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) bool {
		return true
	})
	var v *cmd.VersionOptions
	monkey.PatchInstanceMethod(reflect.TypeOf(v), "UpgradeCli", func(*cmd.VersionOptions) error {
		return errors.New("Returning error for testing UpgradeCli")
	})
}

func TestVersisonCheckWhenCurrentVersionIsGreaterThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	setup(jxVersion)
	version.Map["version"] = "1.4.0"
	opts := &cmd.VersionOptions{}
	err := opts.VersionCheck()
	assert.NoError(t, err, "VersionCheck should exit without failure")
}

func TestVersisonCheckWhenCurrentVersionIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	setup(jxVersion)
	version.Map["version"] = "1.2.3"
	opts := &cmd.VersionOptions{}
	err := opts.VersionCheck()
	assert.NoError(t, err, "VersionCheck should exit without failure")
}

func TestVersisonCheckWhenCurrentVersionIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	setup(jxVersion)
	version.Map["version"] = "1.0.0"
	opts := &cmd.VersionOptions{}
	err := opts.VersionCheck()
	assert.Error(t, err, "VersionCheck should exit with failure")
}

func TestVersisonCheckWhenCurrentVersionIsEqualToReleaseVersionWithPatch(t *testing.T) {
	prVersions := []semver.PRVersion{}
	prVersions = append(prVersions, semver.PRVersion{VersionStr: "dev"})
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3, Pre: prVersions, Build: []string(nil)}
	setup(jxVersion)
	version.Map["version"] = "1.2.3"
	opts := &cmd.VersionOptions{}
	err := opts.VersionCheck()
	assert.NoError(t, err, "VersionCheck should exit without failure")
}

// TODO Would be good to have standardised logging to make testing log output easier...
func TestVersisonCheckWhenCurrentVersionWithPatchIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	setup(jxVersion)
	version.Map["version"] = "1.2.3-dev+6a8285f4"
	opts := &cmd.VersionOptions{}
	err := opts.VersionCheck()
	assert.NoError(t, err, "VersionCheck should exit without failure")
}

func TestVersisonCheckWhenCurrentVersionWithPatchIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	setup(jxVersion)
	version.Map["version"] = "1.2.2-dev+6a8285f4"
	opts := &cmd.VersionOptions{}
	err := opts.VersionCheck()
	assert.NoError(t, err, "VersionCheck should exit without failure")
}
