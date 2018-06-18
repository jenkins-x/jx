package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
)

func (o *CommonOptions) registerLocalHelmRepo(repoName, ns string) error {
	if repoName == "" {
		repoName = kube.LocalHelmRepoName
	}
	// TODO we should use the auth package to keep a list of server login/pwds
	// TODO we have a chartmuseumAuth.yaml now but sure yet if that's the best thing to do
	username := "admin"
	password := "admin"

	// lets check if we have a local helm repository
	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	u, err := kube.FindServiceURL(client, ns, kube.ServiceChartMuseum)
	if err != nil {
		return err
	}
	u2, err := url.Parse(u)
	if err != nil {
		return err
	}
	if u2.User == nil {
		u2.User = url.UserPassword(username, password)
	}
	helmUrl := u2.String()
	// lets check if we already have the helm repo installed or if we need to add it or remove + add it
	text, err := o.getCommandOutput("", "helm", "repo", "list")
	if err != nil {
		return err
	}
	lines := strings.Split(text, "\n")
	remove := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			fields := strings.Fields(t)
			if len(fields) > 1 {
				if fields[0] == repoName {
					if fields[1] == helmUrl {
						return nil
					} else {
						remove = true
					}
				}
			}
		}
	}
	if remove {
		err = o.runCommand("helm", "repo", "remove", repoName)
		if err != nil {
			return err
		}
	}
	return o.runCommand("helm", "repo", "add", repoName, helmUrl)
}

// addHelmRepoIfMissing adds the given helm repo if its not already added
func (o *CommonOptions) addHelmRepoIfMissing(helmUrl string, repoName string) error {
	return o.addHelmBinaryRepoIfMissing("helm", helmUrl, repoName)
}

func (o *CommonOptions) addHelmBinaryRepoIfMissing(helmBinary string, helmUrl string, repoName string) error {
	missing, err := o.isHelmRepoMissing(helmUrl)
	if err != nil {
		return err
	}
	if missing {
		o.Printf("Adding missing helm repo: %s %s\n", util.ColorInfo(repoName), util.ColorInfo(helmUrl))
		err = o.runCommandVerbose(helmBinary, "repo", "add", repoName, helmUrl)
		if err == nil {
			o.Printf("Succesfully added Helm repository %s.\n", repoName)
		}
		return err
	}
	return nil
}

// installChart installs the given chart
func (o *CommonOptions) installChart(releaseName string, chart string, version string, ns string, helmUpdate bool, setValues []string) error {
	return o.installChartAt("", releaseName, chart, version, ns, helmUpdate, setValues)
}

// installChartAt installs the given chart
func (o *CommonOptions) installChartAt(dir string, releaseName string, chart string, version string, ns string, helmUpdate bool, setValues []string) error {
	if helmUpdate {
		o.Printf("Updating Helm repository...\n")
		err := o.runCommand("helm", "repo", "update")
		if err != nil {
			return err
		}
		o.Printf("Helm repository update done.\n")
	}
	timeout := fmt.Sprintf("--timeout=%s", defaultInstallTimeout)
	args := []string{"upgrade", "--install", timeout}
	if version != "" {
		args = append(args, "--version", version)
	}
	if ns != "" {
		kubeClient, _, err := o.KubeClient()
		if err != nil {
			return err
		}
		annotations := map[string]string{"jenkins-x.io/created-by": "Jenkins X"}
		kube.EnsureNamespaceCreated(kubeClient, ns, nil, annotations)
		args = append(args, "--namespace", ns)
	}
	for _, value := range setValues {
		args = append(args, "--set", value)
		o.Printf("Set chart value: --set %s\n", util.ColorInfo(value))
	}
	args = append(args, releaseName, chart)
	return o.runCommandVerboseAt(dir, "helm", args...)
}

// deleteChart deletes the given chart
func (o *CommonOptions) deleteChart(releaseName string, purge bool) error {
	args := []string{"delete"}
	if purge {
		args = append(args, "--purge")
	}
	args = append(args, releaseName)
	return o.runCommandVerbose("helm", args...)
}

func (*CommonOptions) FindHelmChart() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// lets try find the chart file
	chartFile := filepath.Join(dir, "Chart.yaml")
	exists, err := util.FileExists(chartFile)
	if err != nil {
		return "", err
	}
	if !exists {
		// lets try find all the chart files
		files, err := filepath.Glob("*/Chart.yaml")
		if err != nil {
			return "", err
		}
		if len(files) > 0 {
			chartFile = files[0]
		} else {
			files, err = filepath.Glob("*/*/Chart.yaml")
			if err != nil {
				return "", err
			}
			if len(files) > 0 {
				for _, file := range files {
					if !strings.HasSuffix(file, "/preview/Chart.yaml") {
						return file, nil
					}
				}
			}
		}
	}
	return "", nil
}
func (o *CommonOptions) isHelmRepoMissing(helmUrlString string) (bool, error) {
	// lets check if we already have the helm repo installed or if we need to add it or remove + add it
	text, err := o.getCommandOutput("", "helm", "repo", "list")
	if err != nil {
		return false, err
	}
	helmUrl, err := url.Parse(helmUrlString)
	if err != nil {
		return false, err
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			fields := strings.Fields(t)
			if len(fields) > 1 {
				localURL, err := url.Parse(fields[1])
				if err != nil {
					return false, err
				}
				if localURL.Host == helmUrl.Host {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func (o *CommonOptions) addChartRepos(dir string, helmBinary string, chartRepos map[string]string) error {
	installedChartRepos, err := o.getInstalledChartRepos(helmBinary)
	if err != nil {
		return err
	}
	repoCounter := len(installedChartRepos)
	if chartRepos != nil {
		for name, url := range chartRepos {
			if !util.StringMapHasValue(installedChartRepos, url) {
				repoCounter++
				err = o.addHelmBinaryRepoIfMissing(helmBinary, url, name)
				if err != nil {
					return err
				}
			}
		}
	}

	reqfile := filepath.Join(dir, "requirements.yaml")
	exists, err := util.FileExists(reqfile)
	if err != nil {
		return err
	}
	if exists {
		requirements, err := helm.LoadRequirementsFile(reqfile)
		if err != nil {
			return err
		}
		if requirements != nil {
			for _, dep := range requirements.Dependencies {
				repo := dep.Repository
				if repo != "" && !util.StringMapHasValue(installedChartRepos, repo) && repo != defaultChartRepo && !strings.HasPrefix(repo, "file:") {
					repoCounter++
					// TODO we could provide some mechanism to customise the names of repos somehow?
					err = o.addHelmBinaryRepoIfMissing(helmBinary, repo, "repo"+strconv.Itoa(repoCounter))
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (o *CommonOptions) getInstalledChartRepos(helmBinary string) (map[string]string, error) {
	installedChartRepos := map[string]string{}
	text, err := o.getCommandOutput("", helmBinary, "repo", "list")
	if err != nil {
		return installedChartRepos, err
	}
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		if idx > 0 {
			cols := strings.Split(line, "\t")
			if len(cols) > 1 {
				name := strings.TrimSpace(cols[0])
				url := strings.TrimSpace(cols[1])
				if name != "" && url != "" {
					installedChartRepos[name] = url
				}
			}
		}
	}
	return installedChartRepos, nil
}

func (o *CommonOptions) helmInit(dir string) (string, error) {
	helmBinary, err := o.TeamHelmBin()
	if err != nil {
		return helmBinary, err
	}
	o.Printf("Using the helm binary: %s for generating the chart release\n", util.ColorInfo(helmBinary))

	err = o.runCommandVerboseAt(dir, helmBinary, "version")
	if err != nil {
		return helmBinary, err
	}

	if helmBinary == "helm" {
		err = o.runCommandVerboseAt(dir, helmBinary, "init", "--client-only")
	} else {
		err = o.runCommandVerboseAt(dir, helmBinary, "init")
	}
	return helmBinary, err
}

func (o *CommonOptions) helmInitDependencyBuild(dir string, chartRepos map[string]string) (string, error) {
	helmBinary := ""
	path := filepath.Join(dir, "requirements.lock")
	exists, err := util.FileExists(path)
	if err != nil {
		return helmBinary, err
	}
	if exists {
		err = os.Remove(path)
		if err != nil {
			return helmBinary, err
		}
	}
	helmBinary, err = o.TeamHelmBin()
	if err != nil {
		return helmBinary, err
	}
	o.Printf("Using the helm binary: %s for generating the chart release\n", util.ColorInfo(helmBinary))

	err = o.runCommandVerboseAt(dir, helmBinary, "version")
	if err != nil {
		return helmBinary, err
	}

	if helmBinary == "helm" {
		err = o.runCommandVerboseAt(dir, helmBinary, "init", "--client-only")
	} else {
		err = o.runCommandVerboseAt(dir, helmBinary, "init")
	}
	if err != nil {
		return helmBinary, err
	}
	err = o.addChartRepos(dir, helmBinary, chartRepos)
	if err != nil {
		return helmBinary, err
	}

	// TODO due to this issue: https://github.com/kubernetes/helm/issues/4230
	// lets stick with helm2 for this step
	//
	//err = o.runCommandVerboseAt(dir, helmBinary, "dependency", "build", dir)
	err = o.runCommandVerboseAt(dir, "helm", "dependency", "build", dir)
	if err != nil {
		return helmBinary, err
	}

	err = o.runCommandVerboseAt(dir, helmBinary, "lint", dir)
	if err != nil {
		return helmBinary, err
	}
	return helmBinary, nil
}

func (o *CommonOptions) defaultReleaseCharts() map[string]string {
	return map[string]string{
		"releases":  o.releaseChartMuseumUrl(),
		"jenkins-x": "https://chartmuseum.build.cd.jenkins-x.io",
	}
}

func (o *CommonOptions) releaseChartMuseumUrl() string {
	chartRepo := os.Getenv("CHART_REPOSITORY")
	if chartRepo == "" {
		chartRepo = defaultChartRepo
		o.warnf("No $CHART_REPOSITORY defined so using the default value of: %s\n", defaultChartRepo)
	}
	return chartRepo
}
