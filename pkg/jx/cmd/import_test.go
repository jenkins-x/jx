package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

const (
	mavenKeepOldJenkinsfile = "maven-keep-old-jenkinsfile"
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
	_, dirName := filepath.Split(testDir)
	dirName = kube.ToValidName(dirName)
	o := &ImportOptions{}
	configureOptions(&o.CommonOptions)
	o.Dir = testDir
	o.DryRun = true
	o.DisableMaven = true

	if dirName == mavenKeepOldJenkinsfile {
		o.DisableJenkinsfileCheck = true
	}

	err := o.Run()
	assert.NoError(t, err)
	if err == nil {
		jenkinsfile := filepath.Join(testDir, "Jenkinsfile")
		assertFileExists(t, jenkinsfile)
		assertFileExists(t, filepath.Join(testDir, "Dockerfile"))
		assertFileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))

		if dirName == mavenKeepOldJenkinsfile {
			assertFileContains(t, jenkinsfile, "THIS IS OLD!")
			assertFileDoesNotExist(t, jenkinsfile+jenkinsfileBackupSuffix)
		} else {
			if strings.HasPrefix(dirName, "maven") {
				assertFileContains(t, jenkinsfile, "mvn")
			}
			if strings.HasPrefix(dirName, "gradle") {
				assertFileContains(t, jenkinsfile, "gradle")
			}
		}
		if dirName == "maven-old-jenkinsfile" {
			assertFileExists(t, jenkinsfile+jenkinsfileBackupSuffix)
		}
	}
	return err
}

func assertFileContains(t *testing.T, fileName string, containsText string) {
	if assertFileExists(t, fileName) {
		data, err := ioutil.ReadFile(fileName)
		assert.NoError(t, err, "Failed to read file %s", fileName)
		if err == nil {
			text := string(data)
			assert.True(t, strings.Index(text, containsText) >= 0, "The file %s does not contain text: %s", fileName, containsText)
		}
	}
}

func assertFileExists(t *testing.T, fileName string) bool {
	exists, err := util.FileExists(fileName)
	assert.NoError(t, err, "Failed checking if file exists %s", fileName)
	assert.True(t, exists, "File %s should exist", fileName)
	if exists {
		tests.Debugf("File %s exists\n", fileName)
	}
	return exists
}

func assertFileDoesNotExist(t *testing.T, fileName string) bool {
	exists, err := util.FileExists(fileName)
	assert.NoError(t, err, "Failed checking if file exists %s", fileName)
	assert.False(t, exists, "File %s should not exist", fileName)
	if exists {
		tests.Debugf("File %s does not exist\n", fileName)
	}
	return exists
}
