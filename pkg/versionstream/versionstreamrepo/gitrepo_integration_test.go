// +build integration

package versionstreamrepo_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/versionstream/versionstreamrepo"
	"github.com/stretchr/testify/assert"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
)

const (
	RepoURL    = "https://github.com/jenkins-x/jenkins-x-versions"
	VersionRef = "v1.0.114"
	BranchRef  = "master"
	HEAD       = "HEAD"
)

func TestCloneJXVersionsRepoWithDefaultURL(t *testing.T) {
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

	// Get the latest tag so that we know the correct expected verion ref.
	tag, _, err := gitter.Describe(dir, false, VersionRef, "")

	assert.NoError(t, err)
	assert.NotNil(t, dir)
	assert.NotNil(t, versionRef)
	assert.Equal(t, tag, versionRef)
}

func initializeTempGitRepo(gitter gits.Gitter) (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	err = gitter.Init(dir)
	if err != nil {
		return "", err
	}

	err = gitter.AddCommit(dir, "Initial Commit")
	if err != nil {
		return "", err
	}

	err = gitter.CreateTag(dir, VersionRef, "First Tag")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("file://%s", dir), nil
}

func TestCloneJXVersionsRepoWithTeamSettings(t *testing.T) {
	gitter := gits.NewGitCLI()
	gitDir, err := initializeTempGitRepo(gitter)
	defer func() {
		err := os.RemoveAll(gitDir)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	settings := &v1.TeamSettings{
		VersionStreamURL: gitDir,
		VersionStreamRef: VersionRef,
	}
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		"",
		"",
		settings,
		gitter,
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
	gitter := gits.NewGitCLI()
	gitDir, err := initializeTempGitRepo(gitter)
	defer func() {
		err := os.RemoveAll(gitDir)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		gitDir,
		VersionRef,
		nil,
		gitter,
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
	gitter := gits.NewGitCLI()
	gitDir, err := initializeTempGitRepo(gitter)
	defer func() {
		err := os.RemoveAll(gitDir)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		gitDir,
		BranchRef,
		nil,
		gitter,
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
	gitter := gits.NewGitCLI()
	gitDir, err := initializeTempGitRepo(gitter)
	defer func() {
		err := os.RemoveAll(gitDir)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		gitDir,
		HEAD, // We can't know a commit SHA in advance, so this instead of dereferencing it through rev-parse jiggery-pokery
		nil,
		gitter,
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
