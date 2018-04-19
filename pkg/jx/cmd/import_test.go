package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestImportProjects(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-import-projects")
	assert.NoError(t, err)

	testData := path.Join("test_data", "import_projects")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	files, err := ioutil.ReadDir(testData)
	assert.NoError(t, err)

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			srcDir := filepath.Join(testData, name)
			testDir := filepath.Join(tempDir, name)
			util.CopyDir(srcDir, testDir, true)

			err = assertImport(t, testDir)
			assert.NoError(t, err, "Importing dir %s from source %s", testDir, srcDir)
		}
	}
}

func assertImport(t *testing.T, testDir string) error {
	o := &ImportOptions{}
	configureOptions(&o.CommonOptions)
	o.Dir = testDir
	o.DryRun = true
	//_, o.AppName = filepath.Split(testDir)
	err := o.Run()
	assert.NoError(t, err)
	if err == nil {
		_, dirName := filepath.Split(testDir)
		assertFileExists(t, filepath.Join(testDir, "Jenkinsfile"))
		assertFileExists(t, filepath.Join(testDir, "Dockerfile"))
		assertFileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
	}
	return err
}

func assertFileExists(t *testing.T, fileName string) {
	exists, err := util.FileExists(fileName)
	assert.NoError(t, err, "Failed checking if file exists %s", fileName)
	assert.True(t, exists, "File %s should exist", fileName)
	if exists {
		tests.Debugf("File %s exists\n", fileName)
	}
}
