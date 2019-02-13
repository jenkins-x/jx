// +build integration

package cmd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

const (
	gitSuffix                      = "_with_git"
	mavenKeepOldJenkinsfile        = "maven_keep_old_jenkinsfile"
	mavenKeepOldJenkinsfilewithGit = mavenKeepOldJenkinsfile + gitSuffix
	mavenOldJenkinsfile            = "maven_old_jenkinsfile"
	mavenOldJenkinsfilewithGit     = mavenOldJenkinsfile + gitSuffix
	mavenCamel                     = "maven_camel"
	mavenSpringBoot                = "maven_springboot"
	probePrefix                    = "probePath:"
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
			testImportProject(t, tempDir, name, srcDir, false)
			testImportProject(t, tempDir, name, srcDir, true)
		}
	}
}

func testImportProject(t *testing.T, tempDir string, testcase string, srcDir string, withRename bool) {
	testDirSuffix := "DefaultJenkinsfile"
	if withRename {
		testDirSuffix = "RenamedJenkinsfile"
	}
	testDir := filepath.Join(tempDir+"-"+testDirSuffix, testcase)
	util.CopyDir(srcDir, testDir, true)
	if strings.HasSuffix(testcase, gitSuffix) {
		gitDir := filepath.Join(testDir, ".gitdir")
		dotGitExists, gitErr := util.FileExists(gitDir)
		if gitErr != nil {
			log.Warnf("Git source directory %s does not exist: %s", gitDir, gitErr)
		} else if dotGitExists {
			dotGitDir := filepath.Join(testDir, ".git")
			util.RenameDir(gitDir, dotGitDir, true)
		}
	}
	err := assertImport(t, testDir, testcase, withRename)
	assert.NoError(t, err, "Importing dir %s from source %s", testDir, srcDir)
}

func assertImport(t *testing.T, testDir string, testcase string, withRename bool) error {
	_, dirName := filepath.Split(testDir)
	dirName = kube.ToValidName(dirName)
	o := &cmd.ImportOptions{}
	cmd.ConfigureTestOptions(&o.CommonOptions, gits.NewGitCLI(), helm.NewHelmCLI("helm", helm.V2, dirName, true))
	if o.Out == nil {
		o.Out = tests.Output()
	}
	if o.Out == nil {
		o.Out = os.Stdout
	}
	o.Dir = testDir
	o.DryRun = true
	o.DisableMaven = true
	o.LogLevel = "warn"

	if withRename {
		o.Jenkinsfile = "Jenkinsfile-Renamed"
	}

	if strings.HasPrefix(testcase, mavenKeepOldJenkinsfile) {
		o.DisableJenkinsfileCheck = true
	}
	if testcase == mavenCamel || dirName == mavenSpringBoot {
		o.DisableMaven = tests.TestShouldDisableMaven()
	}

	err := o.Run()
	assert.NoError(t, err, "Failed with %s", err)
	if err == nil {
		defaultJenkinsfile := filepath.Join(testDir, jenkinsfile.Name)
		jenkinsfile := defaultJenkinsfile
		if o.Jenkinsfile != "" && o.Jenkinsfile != jenkinsfile.Name {
			jenkinsfile = filepath.Join(testDir, o.Jenkinsfile)
		}
		tests.AssertFileExists(t, jenkinsfile)
		if dirName == "docker" || dirName == "docker-helm" {
			tests.AssertFileExists(t, filepath.Join(testDir, "skaffold.yaml"))
		} else if dirName == "helm" {
			tests.AssertFileDoesNotExist(t, filepath.Join(testDir, "skaffold.yaml"))
		}
		if dirName != "helm" {
			tests.AssertFileExists(t, filepath.Join(testDir, "Dockerfile"))
		} else {
			tests.AssertFileDoesNotExist(t, filepath.Join(testDir, "Dockerfile"))
		}
		if dirName == "docker" {
			tests.AssertFileDoesNotExist(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
			tests.AssertFileDoesNotExist(t, filepath.Join(testDir, "charts"))
			tests.AssertFileDoesNotContain(t, jenkinsfile, "helm")
		} else {
			tests.AssertFileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
		}

		if strings.HasPrefix(testcase, mavenKeepOldJenkinsfile) {
			tests.AssertFileContains(t, jenkinsfile, "THIS IS OLD!")
			tests.AssertFileDoesNotExist(t, jenkinsfile+jenkinsfile.BackupSuffix)
		} else if strings.HasPrefix(testcase, mavenOldJenkinsfile) {
			tests.AssertFileExists(t, jenkinsfile)
			if withRename {
				tests.AssertFileExists(t, defaultJenkinsfile)
				tests.AssertFileContains(t, defaultJenkinsfile, "THIS IS OLD!")
			} else if strings.HasSuffix(testcase, gitSuffix) {
				tests.AssertFileDoesNotExist(t, jenkinsfile+jenkinsfile.BackupSuffix)
			} else {
				tests.AssertFileExists(t, jenkinsfile+jenkinsfile.BackupSuffix)
				tests.AssertFileContains(t, jenkinsfile+jenkinsfile.BackupSuffix, "THIS IS OLD!")
			}
		}
		if strings.HasPrefix(dirName, "maven") && !strings.Contains(testcase, "keep_old") {
			tests.AssertFileContains(t, jenkinsfile, "mvn")
		}
		if strings.HasPrefix(dirName, "gradle") {
			tests.AssertFileContains(t, jenkinsfile, "gradle")
		}

		if !o.DisableMaven {
			if testcase == mavenCamel {
				// should have modified it
				assertProbePathEquals(t, filepath.Join(testDir, "charts", dirName, "values.yaml"), "/health")
			}
			if testcase == mavenSpringBoot {
				// should have left it
				assertProbePathEquals(t, filepath.Join(testDir, "charts", dirName, "values.yaml"), "/actuator/health")
			}
		}
	}
	return err
}

func assertProbePathEquals(t *testing.T, fileName string, expectedProbe string) {
	if tests.AssertFileExists(t, fileName) {
		data, err := ioutil.ReadFile(fileName)
		assert.NoError(t, err, "Failed to read file %s", fileName)
		if err == nil {
			text := string(data)
			found := false
			lines := strings.Split(text, "\n")

			for _, line := range lines {
				if strings.HasPrefix(line, probePrefix) {
					found = true
					value := strings.TrimSpace(strings.TrimPrefix(line, probePrefix))
					assert.Equal(t, expectedProbe, value, "file %s probe with key: %s", fileName, probePrefix)
					break
				}

			}
			assert.True(t, found, "No probe found in file %s with key: %s", fileName, probePrefix)
		}
	}
}
