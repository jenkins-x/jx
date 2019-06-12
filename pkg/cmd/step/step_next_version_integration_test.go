// +build integration

package step_test

import (
	"github.com/jenkins-x/jx/pkg/cmd/step"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestSetVersionJavascript(t *testing.T) {
	originalJxHome, tempJxHome, err := testhelpers.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := testhelpers.CleanupTestJxHomeDir(originalJxHome, tempJxHome)
		assert.NoError(t, err)
	}()
	originalKubeCfg, tempKubeCfg, err := testhelpers.CreateTestKubeConfigDir()
	assert.NoError(t, err)
	defer func() {
		err := testhelpers.CleanupTestKubeConfigDir(originalKubeCfg, tempKubeCfg)
		assert.NoError(t, err)
	}()

	f, err := ioutil.TempDir("", "test-set-version")
	assert.NoError(t, err)

	testData := path.Join("test_data", "next_version", "javascript")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	err = util.CopyDir(testData, f, true)
	assert.NoError(t, err)

	git := gits.NewGitCLI()
	err = git.Init(f)
	assert.NoError(t, err)

	o := step.StepNextVersionOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	o.Out = tests.Output()
	o.Dir = f
	o.Filename = "package.json"
	o.NewVersion = "1.2.3"
	err = o.SetVersion()
	assert.NoError(t, err)

	// root file
	updatedFile, err := util.LoadBytes(o.Dir, o.Filename)
	testFile, err := util.LoadBytes(testData, "expected_package.json")

	assert.Equal(t, string(testFile), string(updatedFile), "replaced version")
}

func TestSetVersionChart(t *testing.T) {
	originalJxHome, tempJxHome, err := testhelpers.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := testhelpers.CleanupTestJxHomeDir(originalJxHome, tempJxHome)
		assert.NoError(t, err)
	}()
	originalKubeCfg, tempKubeCfg, err := testhelpers.CreateTestKubeConfigDir()
	assert.NoError(t, err)
	defer func() {
		err := testhelpers.CleanupTestKubeConfigDir(originalKubeCfg, tempKubeCfg)
		assert.NoError(t, err)
	}()

	f, err := ioutil.TempDir("", "test-set-version")
	assert.NoError(t, err)

	testData := path.Join("test_data", "next_version", "helm")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	err = util.CopyDir(testData, f, true)
	assert.NoError(t, err)

	git := gits.NewGitCLI()
	err = git.Init(f)
	assert.NoError(t, err)

	o := step.StepNextVersionOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	o.Out = tests.Output()
	o.Dir = f
	o.Filename = "Chart.yaml"
	o.NewVersion = "1.2.3"
	err = o.SetVersion()
	assert.NoError(t, err)

	// root file
	updatedFile, err := util.LoadBytes(o.Dir, o.Filename)
	testFile, err := util.LoadBytes(testData, "expected_Chart.yaml")

	assert.Equal(t, string(testFile), string(updatedFile), "replaced version")
}
