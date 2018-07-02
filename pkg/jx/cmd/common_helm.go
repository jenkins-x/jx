package cmd

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
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
	remove := false
	repos, err := o.Helm().ListRepos()
	for repo, repoURL := range repos {
		if repo == repoName {
			if repoURL == helmUrl {
				return nil
			} else {
				remove = true
			}
		}
	}
	if remove {
		err = o.Helm().RemoveRepo(repoName)
		if err != nil {
			return err
		}
	}
	return o.Helm().AddRepo(repoName, helmUrl)
}

// addHelmRepoIfMissing adds the given helm repo if its not already added
func (o *CommonOptions) addHelmRepoIfMissing(helmUrl string, repoName string) error {
	return o.addHelmBinaryRepoIfMissing(helmUrl, repoName)
}

func (o *CommonOptions) addHelmBinaryRepoIfMissing(helmUrl string, repoName string) error {
	missing, err := o.Helm().IsRepoMissing(helmUrl)
	if err != nil {
		return err
	}
	if missing {
		log.Infof("Adding missing helm repo: %s %s\n", util.ColorInfo(repoName), util.ColorInfo(helmUrl))
		err = o.Helm().AddRepo(repoName, helmUrl)
		if err == nil {
			log.Infof("Successfully added Helm repository %s.\n", repoName)
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
		log.Infoln("Updating Helm repository...")
		err := o.Helm().UpdateRepo()
		if err != nil {
			return err
		}
		log.Infoln("Helm repository update done.")
	}
	if ns != "" {
		kubeClient, _, err := o.KubeClient()
		if err != nil {
			return err
		}
		annotations := map[string]string{"jenkins-x.io/created-by": "Jenkins X"}
		kube.EnsureNamespaceCreated(kubeClient, ns, nil, annotations)
	}
	timeout, err := strconv.Atoi(defaultInstallTimeout)
	if err != nil {
		return err
	}
	o.Helm().SetCWD(dir)
	return o.Helm().UpgradeChart(chart, releaseName, ns, &version, true,
		&timeout, false, false, setValues, nil)
}

// deleteChart deletes the given chart
func (o *CommonOptions) deleteChart(releaseName string, purge bool) error {
	return o.Helm().DeleteRelease(releaseName, purge)
}

func (o *CommonOptions) FindHelmChart() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	o.Helm().SetCWD(dir)
	return o.Helm().FindChart()
}

func (o *CommonOptions) isHelmRepoMissing(helmUrlString string) (bool, error) {
	return o.Helm().IsRepoMissing(helmUrlString)
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
				err = o.addHelmBinaryRepoIfMissing(url, name)
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
					err = o.addHelmBinaryRepoIfMissing(repo, "repo"+strconv.Itoa(repoCounter))
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
	return o.Helm().ListRepos()
}

func (o *CommonOptions) helmInit(dir string) error {
	o.Helm().SetCWD(dir)
	_, err := o.Helm().Version()
	if err != nil {
		return err
	}
	if o.Helm().HelmBinary() == "helm" {
		return o.Helm().Init(true, "", "", false)
	} else {
		return o.Helm().Init(false, "", "", false)
	}
}

func (o *CommonOptions) helmInitDependencyBuild(dir string, chartRepos map[string]string) (string, error) {
	o.Helm().SetCWD(dir)
	err := o.Helm().RemoveRequirementsLock()
	if err != nil {
		return o.Helm().HelmBinary(), err
	}

	_, err = o.Helm().Version()
	if err != nil {
		return o.Helm().HelmBinary(), err
	}

	if o.Helm().HelmBinary() == "helm" {
		err = o.Helm().Init(true, "", "", false)
	} else {
		err = o.Helm().Init(false, "", "", false)
	}

	if err != nil {
		return o.Helm().HelmBinary(), err
	}
	err = o.addChartRepos(dir, o.Helm().HelmBinary(), chartRepos)
	if err != nil {
		return o.Helm().HelmBinary(), err
	}

	// TODO due to this issue: https://github.com/kubernetes/helm/issues/4230
	// lets stick with helm2 for this step
	//
	helmBinary := o.Helm().HelmBinary()
	o.Helm().SetHelmBinary("helm")
	o.Helm().SetCWD(dir)
	err = o.Helm().BuildDependency()
	if err != nil {
		return helmBinary, err
	}

	o.Helm().SetHelmBinary(helmBinary)
	_, err = o.Helm().Lint()
	if err != nil {
		return helmBinary, err
	}
	return helmBinary, nil
}

func (o *CommonOptions) defaultReleaseCharts() map[string]string {
	return map[string]string{
		"releases":  o.releaseChartMuseumUrl(),
		"jenkins-x": DEFAULT_CHARTMUSEUM_URL,
	}
}

func (o *CommonOptions) releaseChartMuseumUrl() string {
	chartRepo := os.Getenv("CHART_REPOSITORY")
	if chartRepo == "" {
		chartRepo = defaultChartRepo
		log.Warnf("No $CHART_REPOSITORY defined so using the default value of: %s\n", defaultChartRepo)
	}
	return chartRepo
}
