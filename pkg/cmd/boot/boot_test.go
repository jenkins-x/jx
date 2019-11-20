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

func (o *TestBootOptions) setup(bootRequirements string) {
	o.BootOptions = BootOptions{
		CommonOptions:    &opts.CommonOptions{},
		RequirementsFile: bootRequirements,
	}
}

func TestDetermineGitRef_DefaultGitUrl(t *testing.T) {
	t.Parallel()

	o := TestBootOptions{}
	o.setup(defaultBootRequirements)

	dir := o.createTmpRequirements(t)
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
	o.setup(defaultBootRequirements)

	dir := o.createTmpRequirements(t)
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
	o.setup(defaultBootRequirements)
	o.EnvironmentRepo = url

	dirBefore, err := os.Getwd()
	assert.Nil(t, err, "error should not be nil")
	assert.NotNil(t, dirBefore, "current directory should not be nil")

	err = o.cloneDevEnvironment()
	assert.Nil(t, err, "error should not be nil")
	dirAfter, err := os.Getwd()

	assert.NotEqual(t, dirBefore, dirAfter, "working directory should have been changed")

	assert.Equal(t, dirAfter, filepath.Join(dirBefore, "jenkins-x-boot-config"), "working directory should have jenkins-x-boot-config appended")

	//finally switch back the working dir
	err = os.Chdir(dirBefore)
	assert.Nil(t, err, "error should not be nil")
}

func TestCloneDevEnvironmentIncorrectParam(t *testing.T) {
	t.Parallel()

	o := TestBootOptions{}
	o.setup(defaultBootRequirements)
	o.EnvironmentRepo = "not-a-url"

	err := o.cloneDevEnvironment()

	assert.NotNil(t, err, "error should not be nil")
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
