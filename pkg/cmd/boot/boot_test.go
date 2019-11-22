package boot

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/versionstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBootOptions options for the command
type TestBootOptions struct {
	BootOptions
}

const (
	defaultBootRequirements = "test_data/requirements/jx-requirements.yml"
)

func (o *TestBootOptions) setup(bootRequirements string, dir string) {
	o.BootOptions = BootOptions{
		CommonOptions:    &opts.CommonOptions{},
		RequirementsFile: bootRequirements,
		Dir:              dir,
	}
}

func TestDetermineGitRef_DefaultGitUrl(t *testing.T) {
	t.Parallel()

	o := TestBootOptions{}
	bootDir, err := ioutil.TempDir("", "boot-test")
	require.NoError(t, err)
	defer os.RemoveAll(bootDir)
	o.setup(defaultBootRequirements, bootDir)

	dir := o.createTmpRequirements(t)
	defer os.RemoveAll(dir)
	requirements, _, err := config.LoadRequirementsConfig(dir)
	require.NoError(t, err, "unable to load tmp jx-requirements")
	resolver := &versionstream.VersionResolver{
		VersionsDir: filepath.Join("test_data", "jenkins-x-versions"),
	}

	gitRef, err := o.determineGitRef(resolver, requirements, config.DefaultBootRepository)
	require.NoError(t, err, "unable to determineGitRef")
	assert.Equal(t, "1.0.32", gitRef, "determineGitRef")
}

func TestDetermineGitRef_GitURLNotInVersionStream(t *testing.T) {
	t.Parallel()

	o := TestBootOptions{}
	bootDir, err := ioutil.TempDir("", "boot-test")
	require.NoError(t, err)
	defer os.RemoveAll(bootDir)
	o.setup(defaultBootRequirements, bootDir)

	dir := o.createTmpRequirements(t)
	defer os.RemoveAll(dir)
	requirements, _, err := config.LoadRequirementsConfig(dir)
	require.NoError(t, err, "unable to load tmp jx-requirements")
	resolver := &versionstream.VersionResolver{
		VersionsDir: filepath.Join("test_data", "jenkins-x-versions"),
	}

	gitRef, err := o.determineGitRef(resolver, requirements, "https://github.com/my-fork/my-boot-config.git")
	require.NoError(t, err, "unable to determineGitRef")
	assert.Equal(t, "master", gitRef, "determineGitRef")
}

func TestCloneDevEnvironment(t *testing.T) {

	url := "https://github.com/jenkins-x/jenkins-x-boot-config"
	o := TestBootOptions{}
	bootDir, err := ioutil.TempDir("", "boot-test")
	require.NoError(t, err)
	defer os.RemoveAll(bootDir)
	o.setup(defaultBootRequirements, bootDir)

	cloned, dir, err := o.cloneDevEnvironment(url)
	assert.Nil(t, err, "error should not be nil")

	assert.True(t, cloned)
	assert.Equal(t, "jenkins-x-boot-config", filepath.Base(dir), "cloned dir is incorrect")
}

func TestCloneDevEnvironmentIncorrectParam(t *testing.T) {
	t.Parallel()

	o := TestBootOptions{}
	bootDir, err := ioutil.TempDir("", "boot-test")
	require.NoError(t, err)
	defer os.RemoveAll(bootDir)
	o.setup(defaultBootRequirements, bootDir)
	url := "not-a-url"

	cloned, dir, err := o.cloneDevEnvironment(url)
	assert.NotNil(t, err, "error should not be nil")
	assert.False(t, cloned)
	assert.Empty(t, dir)
}

func (o *TestBootOptions) createTmpRequirements(t *testing.T) string {
	from, err := os.Open(o.RequirementsFile)
	require.NoError(t, err, "unable to open test jx-requirements")

	tmpDir, err := ioutil.TempDir("", "")
	err = os.MkdirAll(tmpDir, util.DefaultWritePermissions)
	to, err := os.Create(filepath.Join(tmpDir, "jx-requirements.yml"))
	require.NoError(t, err, "unable to create tmp jx-requirements")

	_, err = io.Copy(to, from)
	require.NoError(t, err, "unable to copy test jx-requirements to tmp")
	return tmpDir
}
