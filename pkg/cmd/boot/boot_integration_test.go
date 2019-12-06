// +build integration

package boot

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

const (
	FirstTag  = "v1.0.0"
	SecondTag = "v2.0.0"
	ThirdTag  = "v3.0.0"

	testFileName = "some-file"
)

func TestUpdateBootCloneIfOutOfDate_Conflicts(t *testing.T) {
	gitter := gits.NewGitCLI()

	repoDir, err := initializeTempGitRepo(gitter)
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(repoDir)
		assert.NoError(t, err)
	}()

	testDir, err := ioutil.TempDir("", "update-local-boot-clone-test-clone-")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(testDir)
		assert.NoError(t, err)
	}()

	err = gitter.Clone(repoDir, testDir)
	assert.NoError(t, err)

	err = gitter.FetchTags(testDir)
	assert.NoError(t, err)

	err = gitter.Reset(testDir, FirstTag, true)
	assert.NoError(t, err)

	conflictingContent := []byte("something else")
	err = ioutil.WriteFile(filepath.Join(testDir, testFileName), conflictingContent, util.DefaultWritePermissions)
	assert.NoError(t, err)

	o := &BootOptions{
		CommonOptions: &opts.CommonOptions{},
		Dir:           testDir,
	}
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)
	o.CommonOptions.Out = fakeStdout

	err = o.updateBootCloneIfOutOfDate(SecondTag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Could not update local boot clone due to conflicts between local changes and v2.0.0.")

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, "It is based on v1.0.0, but the version stream is using v2.0.0.")
}

func TestUpdateBootCloneIfOutOfDate_NoConflicts(t *testing.T) {
	gitter := gits.NewGitCLI()

	repoDir, err := initializeTempGitRepo(gitter)
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(repoDir)
		assert.NoError(t, err)
	}()

	testDir, err := ioutil.TempDir("", "update-local-boot-clone-test-clone-")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(testDir)
		assert.NoError(t, err)
	}()

	err = gitter.Clone(repoDir, testDir)
	assert.NoError(t, err)

	err = gitter.FetchTags(testDir)
	assert.NoError(t, err)

	err = gitter.Reset(testDir, FirstTag, true)
	assert.NoError(t, err)

	conflictingContent := []byte("something else")
	err = ioutil.WriteFile(filepath.Join(testDir, "some-other-file"), conflictingContent, util.DefaultWritePermissions)
	assert.NoError(t, err)

	o := &BootOptions{
		CommonOptions: &opts.CommonOptions{},
		Dir:           testDir,
	}
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)
	o.CommonOptions.Out = fakeStdout

	err = o.updateBootCloneIfOutOfDate(SecondTag)
	assert.NoError(t, err)

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, "It is based on v1.0.0, but the version stream is using v2.0.0.")
}

func TestUpdateBootCloneIfOutOfDate_UpToDate(t *testing.T) {
	gitter := gits.NewGitCLI()

	repoDir, err := initializeTempGitRepo(gitter)
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(repoDir)
		assert.NoError(t, err)
	}()

	testDir, err := ioutil.TempDir("", "update-local-boot-clone-test-clone-")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(testDir)
		assert.NoError(t, err)
	}()

	err = gitter.Clone(repoDir, testDir)
	assert.NoError(t, err)

	err = gitter.FetchTags(testDir)
	assert.NoError(t, err)

	err = gitter.Reset(testDir, SecondTag, true)
	assert.NoError(t, err)

	conflictingContent := []byte("something else")
	err = ioutil.WriteFile(filepath.Join(testDir, "some-other-file"), conflictingContent, util.DefaultWritePermissions)
	assert.NoError(t, err)

	o := &BootOptions{
		CommonOptions: &opts.CommonOptions{},
		Dir:           testDir,
	}
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)
	o.CommonOptions.Out = fakeStdout

	err = o.updateBootCloneIfOutOfDate(SecondTag)
	assert.NoError(t, err)

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Equal(t, "", output)
}

func TestUpdateBootCloneIfOutOfDate_NotAncestor(t *testing.T) {
	gitter := gits.NewGitCLI()

	repoDir, err := initializeTempGitRepo(gitter)
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(repoDir)
		assert.NoError(t, err)
	}()

	testDir, err := ioutil.TempDir("", "update-local-boot-clone-test-clone-")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(testDir)
		assert.NoError(t, err)
	}()

	err = gitter.Clone(repoDir, testDir)
	assert.NoError(t, err)

	err = gitter.FetchTags(testDir)
	assert.NoError(t, err)

	err = gitter.Reset(testDir, ThirdTag, true)
	assert.NoError(t, err)

	o := &BootOptions{
		CommonOptions: &opts.CommonOptions{},
		Dir:           testDir,
	}
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)
	o.CommonOptions.Out = fakeStdout

	err = o.updateBootCloneIfOutOfDate(SecondTag)
	assert.NoError(t, err)

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, "Current HEAD v3.0.0 in")
}

func initializeTempGitRepo(gitter gits.Gitter) (string, error) {
	dir, err := ioutil.TempDir("", "update-local-boot-clone-test-repo-")
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

	testFile := filepath.Join(dir, testFileName)
	testFileContents := []byte("foo")
	err = ioutil.WriteFile(testFile, testFileContents, util.DefaultWritePermissions)
	if err != nil {
		return "", err
	}

	err = gitter.Add(dir, ".")
	if err != nil {
		return "", err
	}
	err = gitter.AddCommit(dir, "Adding foo")
	if err != nil {
		return "", err
	}

	err = gitter.CreateTag(dir, FirstTag, "First Tag")
	if err != nil {
		return "", err
	}

	testFileContents = []byte("bar")
	err = ioutil.WriteFile(testFile, testFileContents, util.DefaultWritePermissions)
	if err != nil {
		return "", err
	}

	err = gitter.AddCommit(dir, "Adding bar")
	if err != nil {
		return "", err
	}

	err = gitter.CreateTag(dir, SecondTag, "Second Tag")
	if err != nil {
		return "", err
	}

	testFileContents = []byte("baz")
	err = ioutil.WriteFile(testFile, testFileContents, util.DefaultWritePermissions)
	if err != nil {
		return "", err
	}

	err = gitter.AddCommit(dir, "Adding baz")
	if err != nil {
		return "", err
	}

	err = gitter.CreateTag(dir, ThirdTag, "Third Tag")
	if err != nil {
		return "", err
	}

	return dir, nil
}
