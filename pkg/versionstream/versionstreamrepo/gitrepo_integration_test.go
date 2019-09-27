// +build integration

package versionstreamrepo_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/versionstream/versionstreamrepo"
	"github.com/stretchr/testify/assert"
)

const (
	RepoURL    = "https://github.com/jenkins-x/jenkins-x-versions"
	VersionRef = "v1.0.90"
	BranchRef  = "master"
	HEAD       = "HEAD"
)

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
		HEAD,
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
