package gits_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/pborman/uuid"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

type branchNameData struct {
	input    string
	expected string
}

func TestConvertToValidBranchName(t *testing.T) {
	t.Parallel()
	testCases := []branchNameData{
		{
			"testing-thingy", "testing-thingy",
		},
		{
			"testing-thingy/", "testing-thingy",
		},
		{
			"testing-thingy.lock", "testing-thingy",
		},
		{
			"foo bar", "foo_bar",
		},
		{
			"foo\t ~bar", "foo_bar",
		},
	}
	git := &gits.GitCLI{}
	for _, data := range testCases {
		actual := git.ConvertToValidBranchName(data.input)
		assert.Equal(t, data.expected, actual, "Convert to valid branch name for %s", data.input)
	}
}

func TestGitCLI_ShallowCloneNoCommitishOrPr(t *testing.T) {
	t.Parallel()
	gitter := gits.NewGitCLI()
	remoteDir := createLocalGitRepo(t, gitter)
	localDir, err := ioutil.TempDir("", "git-clone")
	assert.NoError(t, err)
	// Add another commit
	generateFileAndAddToGitRepo(t, remoteDir, "master", gitter)
	remoteURL := fmt.Sprintf("file://%s", remoteDir)
	// Do the shallow clone on master
	err = gitter.ShallowClone(localDir, remoteURL, "", "")
	assert.NoError(t, err)
	branch, err := gitter.Branch(localDir)
	assert.NoError(t, err)
	assert.Equal(t, "master", branch)
	commitCount, err := gitCmdWithOutput(localDir, "rev-list", "--count", "master")
	assert.Equal(t, "1", commitCount)
}

func TestGitCLI_ShallowCloneWithPR(t *testing.T) {
	t.Parallel()
	gitter := gits.NewGitCLI()
	remoteDir := createLocalGitRepo(t, gitter)
	localDir, err := ioutil.TempDir("", "git-clone")
	assert.NoError(t, err)
	// Add a commit to PR-1
	name, _ := generateFileAndAddToGitRepo(t, remoteDir, "pull/1/head", gitter)
	remoteURL := fmt.Sprintf("file://%s", remoteDir)
	// Do the shallow clone on master
	err = gitter.ShallowClone(localDir, remoteURL, "", "1")
	assert.NoError(t, err)
	branch, err := gitter.Branch(localDir)
	assert.NoError(t, err)
	assert.Equal(t, "PR-1", branch)
	commitCount, err := gitCmdWithOutput(localDir, "rev-list", "--count", "PR-1")
	assert.NoError(t, err)
	assert.Equal(t, "1", commitCount)
	commitMsg, err := gitter.GetLatestCommitMessage(localDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Add %s", name), commitMsg)
}

func TestGitCLI_ShallowCloneWithCommit(t *testing.T) {
	t.Parallel()
	gitter := gits.NewGitCLI()
	remoteDir := createLocalGitRepo(t, gitter)
	localDir, err := ioutil.TempDir("", "git-clone")
	assert.NoError(t, err)
	// Add a commit to PR-1
	name, commitish := generateFileAndAddToGitRepo(t, remoteDir, uuid.New(), gitter)
	remoteURL := fmt.Sprintf("file://%s", remoteDir)
	// Do the shallow clone on master
	err = gitter.ShallowClone(localDir, remoteURL, commitish, "")
	assert.NoError(t, err)
	branch, err := gitter.Branch(localDir)
	assert.NoError(t, err)
	assert.Equal(t, "master", branch)
	commitCount, err := gitCmdWithOutput(localDir, "rev-list", "--count", "master")
	assert.NoError(t, err)
	assert.Equal(t, "1", commitCount)
	commitMsg, err := gitter.GetLatestCommitMessage(localDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Add %s", name), commitMsg)
}

func TestGitCLI_ShallowCloneWithBranch(t *testing.T) {
	t.Parallel()
	gitter := gits.NewGitCLI()
	remoteDir := createLocalGitRepo(t, gitter)
	localDir, err := ioutil.TempDir("", "git-clone")
	assert.NoError(t, err)
	// Add a commit to PR-1
	branchName := uuid.New()
	name, _ := generateFileAndAddToGitRepo(t, remoteDir, branchName, gitter)
	remoteURL := fmt.Sprintf("file://%s", remoteDir)
	// Do the shallow clone on master
	err = gitter.ShallowClone(localDir, remoteURL, branchName, "")
	assert.NoError(t, err)
	branch, err := gitter.Branch(localDir)
	assert.NoError(t, err)
	assert.Equal(t, branchName, branch)
	commitCount, err := gitCmdWithOutput(localDir, "rev-list", "--count", branchName)
	assert.NoError(t, err)
	assert.Equal(t, "1", commitCount)
	commitMsg, err := gitter.GetLatestCommitMessage(localDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Add %s", name), commitMsg)
}

func TestGitCLI_ShallowCloneWithTag(t *testing.T) {
	gitter := gits.NewVerboseGitCLI()
	remoteDir := createLocalGitRepo(t, gitter)
	localDir, err := ioutil.TempDir("", "git-clone")
	assert.NoError(t, err)
	// Add a commit to master
	name, _ := generateFileAndAddToGitRepo(t, remoteDir, "master", gitter)
	tag := uuid.New()
	// Create a tag
	err = gitter.CreateTag(remoteDir, tag, "test tag")
	assert.NoError(t, err)
	remoteURL := fmt.Sprintf("file://%s", remoteDir)
	// Do the shallow clone of the tag
	err = gitter.ShallowClone(localDir, remoteURL, fmt.Sprintf("tags/%s", tag), "")
	assert.NoError(t, err)
	branch, err := gitter.Branch(localDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("branch-%s", tag), branch)
	commitCount, err := gitCmdWithOutput(localDir, "rev-list", "--count", tag)
	assert.NoError(t, err)
	assert.Equal(t, "1", commitCount)
	commitMsg, err := gitter.GetLatestCommitMessage(localDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Add %s", name), commitMsg)
}

func createLocalGitRepo(t *testing.T, gitter gits.Gitter) string {
	dir, err := ioutil.TempDir("", "git-repo")
	assert.NoError(t, err)
	err = gitter.Init(dir)
	assert.NoError(t, err)
	readmePath := filepath.Join(dir, "README")
	err = ioutil.WriteFile(readmePath, []byte("Test project"), 0655)
	assert.NoError(t, err)
	err = gitter.AddCommit(dir, "Initial commit")
	assert.NoError(t, err)
	return dir
}

func generateFileAndAddToGitRepo(t *testing.T, dir string, branch string, gitter gits.Gitter) (string, string) {
	origBranch, err := gitter.Branch(dir)
	defer func() {
		err := gitter.Checkout(dir, origBranch)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	branches, err := gitter.LocalBranches(dir)
	assert.NoError(t, err)
	found := false
	for _, b := range branches {
		if b == branch {
			found = true
			break
		}
	}
	if !found {
		err := gitter.CreateBranch(dir, branch)
		assert.NoError(t, err)
	}
	err = gitter.Checkout(dir, branch)
	assert.NoError(t, err)
	name := uuid.New()
	path := filepath.Join(dir, name)
	err = ioutil.WriteFile(path, []byte("Test project"), 0655)
	assert.NoError(t, err)
	err = gitter.AddCommit(dir, fmt.Sprintf("Add %s", name))
	assert.NoError(t, err)
	commitish, err := gitter.GetLatestCommitSha(dir)
	assert.NoError(t, err)
	return name, commitish
}

func gitCmdWithOutput(dir string, args ...string) (string, error) {
	cmd := util.Command{
		Dir:  dir,
		Name: "git",
		Args: args,
	}
	return cmd.RunWithoutRetry()
}
