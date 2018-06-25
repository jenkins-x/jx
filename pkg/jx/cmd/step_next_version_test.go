package cmd

import (
	"testing"

	"io/ioutil"
	"os"
	"path"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestMakefile(t *testing.T) {

	o := StepNextVersionOptions{
		Dir:      "test_data/next_version/make",
		Filename: "Makefile",
	}

	v, err := o.getVersion()

	assert.NoError(t, err)

	assert.Equal(t, "1.2.0-SNAPSHOT", v, "error with getVersion for a Makefile")
}

func TestPomXML(t *testing.T) {

	o := StepNextVersionOptions{
		Dir:      "test_data/next_version/java",
		Filename: "pom.xml",
	}

	v, err := o.getVersion()

	assert.NoError(t, err)

	assert.Equal(t, "1.0-SNAPSHOT", v, "error with getVersion for a pom.xml")
}

func TestChart(t *testing.T) {

	o := StepNextVersionOptions{
		Dir:      "test_data/next_version/helm",
		Filename: "Chart.yaml",
	}

	v, err := o.getVersion()

	assert.NoError(t, err)

	assert.Equal(t, "0.0.1-SNAPSHOT", v, "error with getVersion for a pom.xml")
}

func TestSetVersionJavascript(t *testing.T) {
	f, err := ioutil.TempDir("", "test-set-version")
	assert.NoError(t, err)

	testData := path.Join("test_data", "next_version", "javascript")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	err = util.CopyDir(testData, f, true)
	assert.NoError(t, err)

	git := gits.NewGitCLI()
	err = git.GitInit(f)
	assert.NoError(t, err)

	o := StepNextVersionOptions{}
	o.Out = tests.Output()
	o.Dir = f
	o.Filename = "package.json"
	o.NewVersion = "1.2.3"
	err = o.setVersion()
	assert.NoError(t, err)

	// root file
	updatedFile, err := util.LoadBytes(o.Dir, o.Filename)
	testFile, err := util.LoadBytes(testData, "expected_package.json")

	assert.Equal(t, string(testFile), string(updatedFile), "replaced version")

}

func TestSetVersionChart(t *testing.T) {

	f, err := ioutil.TempDir("", "test-set-version")
	assert.NoError(t, err)

	testData := path.Join("test_data", "next_version", "helm")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	err = util.CopyDir(testData, f, true)
	assert.NoError(t, err)

	git := gits.NewGitCLI()
	err = git.GitInit(f)
	assert.NoError(t, err)

	o := StepNextVersionOptions{}
	o.Out = tests.Output()
	o.Dir = f
	o.Filename = "Chart.yaml"
	o.NewVersion = "1.2.3"
	err = o.setVersion()
	assert.NoError(t, err)

	// root file
	updatedFile, err := util.LoadBytes(o.Dir, o.Filename)
	testFile, err := util.LoadBytes(testData, "expected_Chart.yaml")

	assert.Equal(t, string(testFile), string(updatedFile), "replaced version")

}
