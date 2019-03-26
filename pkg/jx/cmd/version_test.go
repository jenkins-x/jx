package cmd_test

import (
	"testing"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestVersisonCheckWhenCurrentVersionIsGreaterThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	version.Map["version"] = "1.4.0"
	opts := &cmd.VersionOptions{
		CommonOptions: &cmd.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersisonCheckWhenCurrentVersionIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.3"
	opts := &cmd.VersionOptions{
		CommonOptions: &cmd.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersisonCheckWhenCurrentVersionIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	version.Map["version"] = "1.0.0"
	opts := &cmd.VersionOptions{
		CommonOptions: &cmd.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.True(t, update, "should update")
}

func TestVersisonCheckWhenCurrentVersionIsEqualToReleaseVersionWithPatch(t *testing.T) {
	prVersions := []semver.PRVersion{}
	prVersions = append(prVersions, semver.PRVersion{VersionStr: "dev"})
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3, Pre: prVersions, Build: []string(nil)}
	version.Map["version"] = "1.2.3"
	opts := &cmd.VersionOptions{
		CommonOptions: &cmd.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersisonCheckWhenCurrentVersionWithPatchIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.3-dev+6a8285f4"
	opts := &cmd.VersionOptions{
		CommonOptions: &cmd.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersisonCheckWhenCurrentVersionWithPatchIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.2-dev+6a8285f4"
	opts := &cmd.VersionOptions{
		CommonOptions: &cmd.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}
