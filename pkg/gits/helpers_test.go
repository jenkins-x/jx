package gits_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/gits"
)

const (
	initialReadme       = "Cheesy!"
	commit1Readme       = "Yet more cheese!"
	commit2Contributing = "Even more cheese!"
	commit3License      = "It's cheesy!"
	contributing        = "CONTRIBUTING"
	readme              = "README"
	license             = "LICENSE"
)

func TestFetchAndMergeOneSHA(t *testing.T) {
	// This test only uses local repos, so it's safe to use real git
	env := prepareFetchAndMergeTests(t)
	defer env.Cleanup()
	// Test merging one commit
	err := gits.FetchAndMergeSHAs([]string{env.Sha1}, "master", env.BaseSha, "origin", env.LocalDir, env.Gitter)
	assert.NoError(t, err)
	readmeFile, err := ioutil.ReadFile(env.ReadmePath)
	assert.NoError(t, err)
	assert.Equal(t, commit1Readme, string(readmeFile))
}

func TestFetchAndMergeMultipleSHAs(t *testing.T) {
	// This test only uses local repos, so it's safe to use real git
	env := prepareFetchAndMergeTests(t)
	defer env.Cleanup()

	// Test merging two commit
	err := gits.FetchAndMergeSHAs([]string{env.Sha1, env.Sha2}, "master", env.BaseSha, "origin", env.LocalDir, env.Gitter)
	assert.NoError(t, err)
	localContributingPath := filepath.Join(env.LocalDir, contributing)
	readmeFile, err := ioutil.ReadFile(env.ReadmePath)
	assert.NoError(t, err)
	assert.Equal(t, commit1Readme, string(readmeFile))
	contributingFile, err := ioutil.ReadFile(localContributingPath)
	assert.NoError(t, err)
	assert.Equal(t, commit2Contributing, string(contributingFile))
}

func TestFetchAndMergeSHAAgainstNonHEADSHA(t *testing.T) {
	// This test only uses local repos, so it's safe to use real git
	env := prepareFetchAndMergeTests(t)
	defer env.Cleanup()

	// Test merging two commit
	err := gits.FetchAndMergeSHAs([]string{env.Sha3}, "master", env.Sha1, "origin", env.LocalDir,
		env.Gitter)
	assert.NoError(t, err)

	readmeFile, err := ioutil.ReadFile(env.ReadmePath)
	assert.NoError(t, err)
	assert.Equal(t, commit1Readme, string(readmeFile))

	localContributingPath := filepath.Join(env.LocalDir, contributing)
	_, err = os.Stat(localContributingPath)
	assert.True(t, os.IsNotExist(err))

	localLicensePath := filepath.Join(env.LocalDir, license)
	licenseFile, err := ioutil.ReadFile(localLicensePath)
	assert.NoError(t, err)
	assert.Equal(t, commit3License, string(licenseFile))
}

type FetchAndMergeTestEnv struct {
	Gitter     *gits.GitCLI
	BaseSha    string
	LocalDir   string
	Sha1       string
	Sha2       string
	Sha3       string
	ReadmePath string
	Cleanup    func()
}

func prepareFetchAndMergeTests(t *testing.T) FetchAndMergeTestEnv {
	gitter := gits.NewGitCLI()

	// Prepare a git repo to test - this is our "remote"
	remoteDir, err := ioutil.TempDir("", "remote")
	assert.NoError(t, err)
	err = gitter.Init(remoteDir)
	assert.NoError(t, err)

	readmePath := filepath.Join(remoteDir, readme)
	contributingPath := filepath.Join(remoteDir, contributing)
	licensePath := filepath.Join(remoteDir, license)
	err = ioutil.WriteFile(readmePath, []byte(initialReadme), 0600)
	assert.NoError(t, err)
	err = gitter.Add(remoteDir, readme)
	assert.NoError(t, err)
	err = gitter.CommitDir(remoteDir, "Initial Commit")
	assert.NoError(t, err)

	// Prepare another git repo, this is local repo
	localDir, err := ioutil.TempDir("", "local")
	assert.NoError(t, err)
	err = gitter.Init(localDir)
	assert.NoError(t, err)
	// Set up the remote
	err = gitter.AddRemote(localDir, "origin", remoteDir)
	assert.NoError(t, err)
	err = gitter.FetchBranch(localDir, "origin", "master")
	assert.NoError(t, err)
	err = gitter.Merge(localDir, "origin/master")
	assert.NoError(t, err)

	localReadmePath := filepath.Join(localDir, readme)
	readmeFile, err := ioutil.ReadFile(localReadmePath)
	assert.NoError(t, err)
	assert.Equal(t, initialReadme, string(readmeFile))
	baseSha, err := gitter.GetLatestCommitSha(localDir)
	assert.NoError(t, err)

	err = ioutil.WriteFile(readmePath, []byte(commit1Readme), 0600)
	assert.NoError(t, err)
	err = gitter.Add(remoteDir, readme)
	assert.NoError(t, err)
	err = gitter.CommitDir(remoteDir, "More Cheese")
	assert.NoError(t, err)
	sha1, err := gitter.GetLatestCommitSha(remoteDir)
	assert.NoError(t, err)

	err = ioutil.WriteFile(contributingPath, []byte(commit2Contributing), 0600)
	assert.NoError(t, err)
	err = gitter.Add(remoteDir, contributing)
	assert.NoError(t, err)
	err = gitter.CommitDir(remoteDir, "Even More Cheese")
	assert.NoError(t, err)
	sha2, err := gitter.GetLatestCommitSha(remoteDir)
	assert.NoError(t, err)

	// Put some commits on a branch
	branchName := uuid.NewV4().String()
	err = gitter.CreateBranchFrom(remoteDir, branchName, baseSha)
	assert.NoError(t, err)
	err = gitter.Checkout(remoteDir, branchName)
	assert.NoError(t, err)

	err = ioutil.WriteFile(licensePath, []byte(commit3License), 0600)
	assert.NoError(t, err)
	err = gitter.Add(remoteDir, license)
	assert.NoError(t, err)
	err = gitter.CommitDir(remoteDir, "Even More Cheese")
	assert.NoError(t, err)
	sha3, err := gitter.GetLatestCommitSha(remoteDir)
	assert.NoError(t, err)

	return FetchAndMergeTestEnv{
		Gitter:     gitter,
		BaseSha:    baseSha,
		LocalDir:   localDir,
		Sha1:       sha1,
		Sha2:       sha2,
		Sha3:       sha3,
		ReadmePath: localReadmePath,
		Cleanup: func() {
			err := os.RemoveAll(localDir)
			assert.NoError(t, err)
			err = os.RemoveAll(remoteDir)
			assert.NoError(t, err)
		},
	}
}
