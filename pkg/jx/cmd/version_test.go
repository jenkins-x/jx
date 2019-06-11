package cmd_test

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/testhelpers"
	"path"
	"testing"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestVersisonCheckWhenCurrentVersionIsGreaterThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	version.Map["version"] = "1.4.0"
	opts := &cmd.VersionOptions{
		CommonOptions: &opts.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersisonCheckWhenCurrentVersionIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.3"
	opts := &cmd.VersionOptions{
		CommonOptions: &opts.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersisonCheckWhenCurrentVersionIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	version.Map["version"] = "1.0.0"
	opts := &cmd.VersionOptions{
		CommonOptions: &opts.CommonOptions{},
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
		CommonOptions: &opts.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersisonCheckWhenCurrentVersionWithPatchIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.3-dev+6a8285f4"
	opts := &cmd.VersionOptions{
		CommonOptions: &opts.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersisonCheckWhenCurrentVersionWithPatchIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.2-dev+6a8285f4"
	opts := &cmd.VersionOptions{
		CommonOptions: &opts.CommonOptions{},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestDockerImageGetsLabel(t *testing.T) {
	t.Parallel()

	versionsDir := path.Join("test_data", "common_versions")
	assert.DirExists(t, versionsDir)

	o := &opts.CommonOptions{}
	testhelpers.ConfigureTestOptions(o, gits.NewGitCLI(), helm.NewHelmCLI("helm", helm.V2, "", true))

	resolver := &opts.VersionResolver{
		VersionsDir: versionsDir,
	}

	testData := map[string]string{
		"alreadyversioned:7.8.9": "alreadyversioned:7.8.9",
		"maven":                  "maven:1.2.3",
		"docker.io/maven":        "maven:1.2.3",
		"gcr.io/cheese":          "gcr.io/cheese:4.5.6",
		"noversion":              "noversion",
	}

	for image, expected := range testData {
		actual, err := resolver.ResolveDockerImage(image)
		if assert.NoError(t, err, "resolving image %s", image) {
			assert.Equal(t, expected, actual, "resolving image %s", image)
		}
	}
}
