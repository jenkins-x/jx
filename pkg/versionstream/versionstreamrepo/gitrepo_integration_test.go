// +build integration

package versionstreamrepo_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/versionstream/versionstreamrepo"
	"github.com/stretchr/testify/assert"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
)

const (
	RepoURL    = "https://github.com/jenkins-x/jenkins-x-versions-test.git"
	VersionRef = "v1.0.103"
	BranchRef  = "master"
	HEAD       = "HEAD"
	CommitRef  = "ea16897b7b12f08708c307de49ff21b2a197517c"
)

func TestCloneJXVersionsRepoWithNoURL(t *testing.T) {
	gitter := gits.NewGitCLI()
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		"",
		VersionRef,
		nil,
		gitter,
		true,
		false,
		nil,
		nil,
		nil,
	)

	// Pull the latest tag so that we know the correct expected verion ref.
	tag, _, err := gitter.Describe(dir, false, VersionRef, "")

	assert.NoError(t, err)
	assert.NotNil(t, dir)
	assert.NotNil(t, versionRef)
	assert.Equal(t, tag, versionRef)
}

func TestCloneJXVersionsRepoWithTeamSettings(t *testing.T) {
	settings := &v1.TeamSettings{
		VersionStreamURL: RepoURL,
		VersionStreamRef: VersionRef,
	}
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		"",
		"",
		settings,
		gits.NewGitCLI(),
		true,
		false,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.NotNil(t, dir)
	assert.NotNil(t, versionRef)
	assert.Equal(t, VersionRef, versionRef)
}

func TestCloneJXVersionsRepoWithATag(t *testing.T) {
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		RepoURL,
		VersionRef,
		nil,
		gits.NewGitCLI(),
		true,
		false,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.NotNil(t, dir)
	assert.NotNil(t, versionRef)
	assert.Equal(t, VersionRef, versionRef)
}

func TestCloneJXVersionsRepoWithABranch(t *testing.T) {
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		RepoURL,
		BranchRef,
		nil,
		gits.NewGitCLI(),
		true,
		false,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.NotNil(t, dir)
	assert.NotNil(t, versionRef)
	assert.Equal(t, VersionRef, versionRef)
}

func TestCloneJXVersionsRepoWithACommit(t *testing.T) {
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		RepoURL,
		CommitRef,
		nil,
		gits.NewGitCLI(),
		true,
		false,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.NotNil(t, dir)
	assert.NotNil(t, versionRef)
	assert.Equal(t, VersionRef, versionRef)
}
