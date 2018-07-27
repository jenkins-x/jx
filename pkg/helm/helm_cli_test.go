package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bouk/monkey"
	"github.com/jenkins-x/jx/pkg/util"
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
const listReleasesOutput = `
NAME                            REVISION        UPDATED                         STATUS          CHART                           NAMESPACE
jenkins-x                       1               Mon Jul  2 16:16:20 2018        DEPLOYED        jenkins-x-platform-0.0.1655     jx
jx-production                   1               Mon Jul  2 16:22:47 2018        DEPLOYED        env-0.0.1                       jx-production
jx-staging                      1               Mon Jul  2 16:21:06 2018        DEPLOYED        env-0.0.1                       jx-staging
jxing                           1               Wed Jun  6 14:24:42 2018        DEPLOYED        nginx-ingress-0.20.1            kube-system
vault-operator                  1               Mon Jun 25 16:09:28 2018        DEPLOYED        vault-operator-0.1.0            jx
`

func checkArgs(cli *HelmCLI, expectedDir string, expectedName string, exptectedArgs string) error {
	if cli.Runner.Dir != expectedDir {
		return fmt.Errorf("expected directory '%s', got: '%s'", expectedDir, cli.Runner.Dir)
	}
	if cli.Runner.Name != expectedName {
		return fmt.Errorf("expected binary: '%s', got: '%s'", expectedName, cli.Runner.Name)
	}
	argsStr := strings.Join(cli.Runner.Args, " ")
	if argsStr != exptectedArgs {
		return fmt.Errorf("expected args: '%s', go: '%s'", exptectedArgs, argsStr)
	}
	return nil
}

func setup(output string) {
	var r *util.Command // Has to be a pointer to because `RunWithoutRetry` has a pointer receiver
	monkey.PatchInstanceMethod(reflect.TypeOf(r), "RunWithoutRetry", func(_ *util.Command) (string, error) {
		return output, nil
	})
}

func createHelm(expectedArgs string) (*HelmCLI, error) {
	cli := NewHelmCLI(binary, V2, cwd, expectedArgs)
	err := checkArgs(cli, cwd, binary, expectedArgs)
	return cli, err
}

func TestNewHelmCLI(t *testing.T) {
	setup("")
	cli := NewHelmCLI(binary, V2, cwd, "arg1 arg2 arg3", "and some", "more")
	assert.Equal(t, []string{"arg1", "arg2", "arg3", "and", "some", "more"}, cli.Runner.Args)

	cli, _ = createHelm("arg1 arg2 arg3")
	assert.Equal(t, binary, cli.Binary)
	assert.Equal(t, cwd, cli.CWD)
	assert.Equal(t, binary, cli.Runner.Name)
	assert.Equal(t, cwd, cli.Runner.Dir)
	assert.Equal(t, []string{"arg1", "arg2", "arg3"}, cli.Runner.Args)
}

func TestInit(t *testing.T) {
	setup("")
	expectedArgs := fmt.Sprintf("init --client-only --service-account %s --tiller-namespace %s --upgrade",
		serviceAccount, namespace)
	helm, err := createHelm(expectedArgs)

	assert.NoError(t, err, "should create helm without any error")
	err = helm.Init(true, serviceAccount, namespace, true)
	assert.NoError(t, err, "should init helm without any error")
}

func TestAddRepo(t *testing.T) {
	setup("")
	expectedArgs := fmt.Sprintf("repo add %s %s", repo, repoURL)
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.AddRepo(repo, repoURL)
	assert.NoError(t, err, "should add helm repo without any error")
}
func TestRemoveRepo(t *testing.T) {
	setup("")
	expectedArgs := fmt.Sprintf("repo remove %s", repo)
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.RemoveRepo(repo)
	assert.NoError(t, err, "should remove helm repo without any error")
}

func TestListRepos(t *testing.T) {
	setup(listRepoOutput)
	expectedArgs := "repo list"
	helm, err := createHelm(expectedArgs)
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
	setup(listRepoOutput)
	expectedArgs := "repo list"
	helm, _ := createHelm(expectedArgs)
	url := "https://chartmuseum.build.cd.jenkins-x.io"
	missing, err := helm.IsRepoMissing(url)
	assert.NoError(t, err, "should search missing repos without any error")
	assert.False(t, missing, "should find url '%s'", url)
	url = "https://test"
	missing, err = helm.IsRepoMissing(url)
	assert.NoError(t, err, "search missing repos should not return an error")
	assert.True(t, missing, "should not find url '%s'", url)
}

func TestUpdateRepo(t *testing.T) {
	setup("")
	expectedArgs := "repo update"
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.UpdateRepo()
	assert.NoError(t, err, "should update  helm repo without any error")
}

func TestRemoveRequirementsLock(t *testing.T) {
	setup("")
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
	setup("")
	expectedArgs := "dependency build"
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.BuildDependency()
	assert.NoError(t, err, "should build helm repo dependencies without any error")
}

func TestInstallChart(t *testing.T) {
	setup("")
	value := []string{"test"}
	valueFile := []string{"./myvalues.yaml"}
	expectedArgs := fmt.Sprintf("install --name %s --namespace %s %s --set %s --values %s",
		releaseName, namespace, chart, value[0], valueFile[0])
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.InstallChart(chart, releaseName, namespace, nil, nil, value, valueFile)
	assert.NoError(t, err, "should install the chart without any error")
}

func TestUpgradeChart(t *testing.T) {
	setup("")
	value := []string{"test"}
	valueFile := []string{"./myvalues.yaml"}
	version := "0.0.1"
	timeout := 600
	expectedArgs := fmt.Sprintf("upgrade --namespace %s --install --wait --force --timeout %d --version %s --set %s --values %s %s %s",
		namespace, timeout, version, value[0], valueFile[0], releaseName, chart)
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.UpgradeChart(chart, releaseName, namespace, &version, true, &timeout, true, true, value, valueFile)
	assert.NoError(t, err, "should upgrade the chart without any error")
}

func TestDeleteRelaese(t *testing.T) {
	setup("")
	expectedArgs := fmt.Sprintf("delete --purge %s", releaseName)
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.DeleteRelease(releaseName, true)
	assert.NoError(t, err, "should delete helm chart release without any error")
}

func TestStatusRelease(t *testing.T) {
	setup("")
	expectedArgs := fmt.Sprintf("status %s", releaseName)
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.StatusRelease(releaseName)
	assert.NoError(t, err, "should get the status of a helm chart release without any error")
}

func TestStatusReleases(t *testing.T) {
	setup(listReleasesOutput)
	expectedArgs := "list"
	expectedSatusMap := map[string]string{
		"jenkins-x":      "DEPLOYED",
		"jx-production":  "DEPLOYED",
		"jx-staging":     "DEPLOYED",
		"jxing":          "DEPLOYED",
		"vault-operator": "DEPLOYED",
	}
	helm, _ := createHelm(expectedArgs)
	statusMap, err := helm.StatusReleases()
	assert.NoError(t, err, "should list the release statuses without any error")
	for release, status := range statusMap {
		assert.Equal(t, expectedSatusMap[release], status, "expected status '%s', got '%s'", expectedSatusMap[release], status)
	}
}

func TestLint(t *testing.T) {
	expectedArgs := "lint"
	expectedOutput := "test"
	setup(expectedOutput)
	helm, _ := createHelm(expectedArgs)
	output, err := helm.Lint()
	assert.NoError(t, err, "should lint the chart without any error")
	assert.Equal(t, "test", output)
}

func TestVersion(t *testing.T) {
	expectedArgs := "version --short"
	expectedOutput := "1.0.0"
	setup(expectedOutput)
	helm, _ := createHelm(expectedArgs)
	output, err := helm.Version(false)
	assert.NoError(t, err, "should get the version without any error")
	assert.Equal(t, expectedOutput, output)
}

func TestSearchChartVersions(t *testing.T) {
	expectedOutput := searchVersionOutput
	expectedArgs := fmt.Sprintf("search %s --versions", chart)
	setup(expectedOutput)
	helm, _ := createHelm(expectedArgs)
	versions, err := helm.SearchChartVersions(chart)
	assert.NoError(t, err, "should search chart versions without any error")
	expectedVersions := []string{"0.0.1481", "0.0.1480", "0.0.1479"}
	for i, version := range versions {
		assert.Equal(t, expectedVersions[i], version, "should parse the version '%s'", version)
	}
}

func TestFindChart(t *testing.T) {
	setup("")
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

func TestPackage(t *testing.T) {
	setup("")
	expectedArgs := fmt.Sprintf("package %s", cwd)
	helm, err := createHelm(expectedArgs)
	assert.NoError(t, err, "should create helm without any error")
	err = helm.PackageChart()
	assert.NoError(t, err, "should package chart without any error")
}
