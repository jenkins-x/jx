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
	RepoURL           = "https://github.com/jenkins-x/jenkins-x-versions"
	TagFromDefaultURL = "v1.0.114"
	FirstTag          = "v0.0.1"
	SecondTag         = "v0.02"
	BranchRef         = "master"
	HEAD              = "HEAD"
)

func TestCloneJXVersionsRepoWithDefaultURL(t *testing.T) {
	gitter := gits.NewGitCLI()
	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		"",
		TagFromDefaultURL,
		nil,
		gitter,
		true,
		false,
		nil,
		nil,
		nil,
	)

	// Get the latest tag so that we know the correct expected verion ref.
	tag, _, err := gitter.Describe(dir, false, TagFromDefaultURL, "")

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

	err = gitter.CreateTag(dir, FirstTag, "First Tag")
	if err != nil {
		return "", err
	}

	return fmt.Sprint(dir), nil
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
		VersionStreamRef: FirstTag,
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
	assert.Equal(t, FirstTag, versionRef)
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
		FirstTag,
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
	assert.Equal(t, FirstTag, versionRef)
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
	assert.Equal(t, FirstTag, versionRef)
}

func TestCloneJXVersionsRepoWithACommit(t *testing.T) {
	gitter := gits.NewGitCLI()
	gitDir, err := initializeTempGitRepo(gitter)
	defer func() {
		err := os.RemoveAll(gitDir)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	testFile, err := ioutil.TempFile(gitDir, "versionstreams-test-")
	assert.NoError(t, err)
	defer os.Remove(testFile.Name())

	testFileContents := []byte("foo")
	_, err = testFile.Write(testFileContents)
	assert.NoError(t, err)

	err = gitter.AddCommit(gitDir, "Adding foo")
	assert.NoError(t, err)

	testFileContents = []byte("bar")
	_, err = testFile.Write(testFileContents)
	assert.NoError(t, err)

	err = gitter.AddCommit(gitDir, "Adding bar")
	assert.NoError(t, err)

	err = gitter.CreateTag(gitDir, SecondTag, "Second Tag")
	assert.NoError(t, err)

	headMinusOne, err := gitter.RevParse(gitDir, "HEAD~1")

	dir, versionRef, err := versionstreamrepo.CloneJXVersionsRepo(
		fmt.Sprintf("file://%s", gitDir),
		headMinusOne,
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
	assert.Equal(t, SecondTag, versionRef)

	err = gitter.Checkout(dir, versionRef)
	assert.NoError(t, err)

	actualFileContents, err := ioutil.ReadFile(testFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, []byte("foobar"), actualFileContents)
}
