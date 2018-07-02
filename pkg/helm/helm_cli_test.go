package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const binary = "helm"
const cwd = "test"
const repo = "test-repo"
const repoURL = "http://test-repo"
const serviceAccount = "test-sa"
const namespace = "test-namespace"
const chart = "test-chart"
const releaseName = "test-release"
const listRepoOutput = `
NAME            URL
stable          https://kubernetes-charts.storage.googleapis.com
local           http://127.0.0.1:8879/charts
jenkins-x       https://chartmuseum.build.cd.jenkins-x.io
	`
const searchVersionOutput = `
NAME                                    CHART VERSION   APP VERSION     DESCRIPTION
jenkins-x/jenkins-x-platform            0.0.1481                        Jenkins X 
jenkins-x/jenkins-x-platform            0.0.1480                        Jenkins X 
jenkins-x/jenkins-x-platform            0.0.1479                        Jenkins X 
`

func checkArgs(expectedDir string, expectedName string, exptectedArgs string, dir string, name string, args ...string) error {
	if dir != expectedDir {
		return fmt.Errorf("expected directory '%s', got: '%s'", expectedDir, dir)
	}
	if name != expectedName {
		return fmt.Errorf("expected binary: '%s', got: '%s'", binary, name)
	}
	argsStr := strings.Join(args, " ")
	if argsStr != exptectedArgs {
		return fmt.Errorf("expected args: '%s', go: '%s'", exptectedArgs, argsStr)
	}
	return nil
}

func createHelm(expectedArgs string) *HelmCLI {
	return newHelmCLIWithRunner(binary, V2, cwd, helmRunner{
		run: func(dir string, name string, args ...string) error {
			return checkArgs(cwd, binary, expectedArgs, dir, name, args...)
		},
		runWithOutput: nil,
	})
}

func createHelmWithOutput(expectedArgs string, output string) *HelmCLI {
	return newHelmCLIWithRunner(binary, V2, cwd, helmRunner{
		run: nil,
		runWithOutput: func(dir string, name string, args ...string) (string, error) {
			err := checkArgs(cwd, binary, expectedArgs, dir, name, args...)
			if err != nil {
				return "", err
			}
			return output, nil
		},
	})
}

func TestInit(t *testing.T) {
	expectedArgs := fmt.Sprintf("init --client-only --service-account %s --tiller-namespace %s --upgrade",
		serviceAccount, namespace)
	helm := createHelm(expectedArgs)
	err := helm.Init(true, serviceAccount, namespace, true)
	assert.NoError(t, err, "should init helm without any error")

}

func TestAddRepo(t *testing.T) {
	expectedArgs := fmt.Sprintf("repo add %s %s", repo, repoURL)
	helm := createHelm(expectedArgs)
	err := helm.AddRepo(repo, repoURL)
	assert.NoError(t, err, "should add helm repo without any error")
}
func TestRemoveRepo(t *testing.T) {
	expectedArgs := fmt.Sprintf("repo remove %s", repo)
	helm := createHelm(expectedArgs)
	err := helm.RemoveRepo(repo)
	assert.NoError(t, err, "should remove helm repo without any error")
}

func TestListRepos(t *testing.T) {
	expectedArgs := "repo list"
	output := listRepoOutput
	helm := createHelmWithOutput(expectedArgs, output)
	repos, err := helm.ListRepos()
	assert.NoError(t, err, "should list helm repos without any error")

	expectedRepos := map[string]string{
		"stable":    "https://kubernetes-charts.storage.googleapis.com",
		"local":     "http://127.0.0.1:8879/charts",
		"jenkins-x": "https://chartmuseum.build.cd.jenkins-x.io",
	}
	assert.Equal(t, len(expectedRepos), len(repos), "should list the same number of repos")
	for k, v := range repos {
		assert.Equal(t, expectedRepos[k], v, "should parse correctly the repo URL")
	}
}

func TestIsRepoMissing(t *testing.T) {
	expectedArgs := "repo list"
	output := listRepoOutput
	helm := createHelmWithOutput(expectedArgs, output)
	url := "https://chartmuseum.build.cd.jenkins-x.io"
	missing, err := helm.IsRepoMissing(url)
	assert.NoError(t, err, "should search missing repos without any error")
	assert.False(t, missing, "should find url '%s'", url)
	url = "https://test"
	missing, err = helm.IsRepoMissing(url)
	assert.Error(t, err, "should search missing repos return an error")
	assert.True(t, missing, "should not find url '%s'", url)
}

func TestUpdateRepo(t *testing.T) {
	expectedArgs := "repo update"
	helm := createHelm(expectedArgs)
	err := helm.UpdateRepo()
	assert.NoError(t, err, "should update  helm repo without any error")
}

func TestRemoveRequirementsLock(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "reqtest")
	assert.NoError(t, err, "should be able to create a temporary dir")
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "requirements.lock")
	ioutil.WriteFile(path, []byte("test"), 0644)
	helm := &HelmCLI{
		CWD: dir,
	}
	err = helm.RemoveRequirementsLock()
	assert.NoError(t, err, "should remove requirements.lock file")
}

func TestBuildDependency(t *testing.T) {
	expectedArgs := "dependency build"
	helm := createHelm(expectedArgs)
	err := helm.BuildDependency()
	assert.NoError(t, err, "should build helm repo dependencies without any error")
}

func TestInstallChart(t *testing.T) {
	value := []string{"test"}
	valueFile := []string{"./myvalues.yaml"}
	expectedArgs := fmt.Sprintf("install --name %s --namespace %s %s --set %s --values %s",
		releaseName, namespace, chart, value[0], valueFile[0])
	helm := createHelm(expectedArgs)
	err := helm.InstallChart(chart, releaseName, namespace, nil, nil, value, valueFile)
	assert.NoError(t, err, "should install the chart without any error")
}

func TestUpgradeChart(t *testing.T) {
	value := []string{"test"}
	valueFile := []string{"./myvalues.yaml"}
	version := "0.0.1"
	timeout := 600
	expectedArgs := fmt.Sprintf("upgrade --namespace %s --install --wait --force --timeout %d --version %s --set %s --values %s %s %s",
		namespace, timeout, version, value[0], valueFile[0], releaseName, chart)
	helm := createHelm(expectedArgs)
	err := helm.UpgradeChart(chart, releaseName, namespace, &version, true, &timeout, true, true, value, valueFile)
	assert.NoError(t, err, "should upgrade the chart without any error")
}

func TestDeleteRelaese(t *testing.T) {
	expectedArgs := fmt.Sprintf("delete --purge %s", releaseName)
	helm := createHelm(expectedArgs)
	err := helm.DeleteRelease(releaseName, true)
	assert.NoError(t, err, "should delete helm chart release without any error")
}

func TestStatus(t *testing.T) {
	expectedArgs := fmt.Sprintf("status %s", releaseName)
	helm := createHelm(expectedArgs)
	err := helm.StatusRelease(releaseName)
	assert.NoError(t, err, "should get the status of a helm chart release without any error")
}

func TestLint(t *testing.T) {
	expectedArgs := "lint"
	expectedOutput := "test"
	helm := createHelmWithOutput(expectedArgs, expectedOutput)
	output, err := helm.Lint()
	assert.NoError(t, err, "should lint the chart without any error")
	assert.Equal(t, expectedOutput, output)
}

func TestVersion(t *testing.T) {
	expectedArgs := "version"
	expectedOutput := "1.0.0"
	helm := createHelmWithOutput(expectedArgs, expectedOutput)
	output, err := helm.Version()
	assert.NoError(t, err, "should get the version without any error")
	assert.Equal(t, expectedOutput, output)
}

func TestSearchChartVersions(t *testing.T) {
	expectedOutput := searchVersionOutput
	expectedArgs := fmt.Sprintf("search %s --versions", chart)
	helm := createHelmWithOutput(expectedArgs, expectedOutput)
	versions, err := helm.SearchChartVersions(chart)
	assert.NoError(t, err, "should search chart versions without any error")
	expectedVersions := []string{"0.0.1481", "0.0.1480", "0.0.1479"}
	for i, version := range versions {
		assert.Equal(t, expectedVersions[i], version, "should parse the version '%s'", version)
	}
}

func TestFindChart(t *testing.T) {
	chartFile := "Chart.yaml"
	dir, err := ioutil.TempDir("/tmp", "charttest")
	assert.NoError(t, err, "should be able to create a temporary dir")
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, chartFile)
	ioutil.WriteFile(path, []byte("test"), 0644)
	helm := &HelmCLI{
		CWD: dir,
	}
	chartFile, err = helm.FindChart()
	assert.NoError(t, err, "should find the chart file")
	assert.Equal(t, path, chartFile, "should find chart file '%s'", path)
}
