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
	"github.com/pkg/errors"
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
		return errors.Wrap(err, "failed to create the kube client")
	}
	u, err := kube.FindServiceURL(client, ns, kube.ServiceChartMuseum)
	if err != nil {
		return errors.Wrapf(err, "failed to find the service URL of the chartmuseum")
	}
	u2, err := url.Parse(u)
	if err != nil {
		return errors.Wrap(err, "failed to parse the chartmuseum URL")
	}
	if u2.User == nil {
		u2.User = url.UserPassword(username, password)
	}
	helmUrl := u2.String()
	// lets check if we already have the helm repo installed or if we need to add it or remove + add it
	remove := false
	repos, err := o.Helm().ListRepos()
	if err != nil {
		return errors.Wrap(err, "failed to list the repositories")
	}
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
			return errors.Wrapf(err, "failed to remove the repository '%s'", repoName)
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
		return errors.Wrapf(err, "failed to check if the repository with URL '%s' is missing", helmUrl)
	}
	if missing {
		log.Infof("Adding missing helm repo: %s %s\n", util.ColorInfo(repoName), util.ColorInfo(helmUrl))
		err = o.Helm().AddRepo(repoName, helmUrl)
		if err == nil {
			log.Infof("Successfully added Helm repository %s.\n", repoName)
		}
		return errors.Wrapf(err, "failed to add the repository '%s' with URL '%s'", repoName, helmUrl)
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
			return errors.Wrap(err, "failed to update repository")
		}
		log.Infoln("Helm repository update done.")
	}
	if ns != "" {
		kubeClient, _, err := o.KubeClient()
		if err != nil {
			return errors.Wrap(err, "failed to create the kube client")
		}
		annotations := map[string]string{"jenkins-x.io/created-by": "Jenkins X"}
		kube.EnsureNamespaceCreated(kubeClient, ns, nil, annotations)
	}
	timeout, err := strconv.Atoi(defaultInstallTimeout)
	if err != nil {
		return errors.Wrap(err, "failed to convert the timeout to an int")
	}
	o.Helm().SetCWD(dir)
	return o.Helm().UpgradeChart(chart, releaseName, ns, &version, true,
		&timeout, true, false, setValues, nil)
}

// deleteChart deletes the given chart
func (o *CommonOptions) deleteChart(releaseName string, purge bool) error {
	return o.Helm().DeleteRelease(releaseName, purge)
}

func (o *CommonOptions) FindHelmChart() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get the current working directory")
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
		return errors.Wrap(err, "failed to retrieve the install charts")
	}
	repoCounter := len(installedChartRepos)
	if chartRepos != nil {
		for name, url := range chartRepos {
			if !util.StringMapHasValue(installedChartRepos, url) {
				repoCounter++
				err = o.addHelmBinaryRepoIfMissing(url, name)
				if err != nil {
					return errors.Wrapf(err, "failed to add the helm repository with name '%s' and URL '%s'", name, url)
				}
			}
		}
	}

	reqfile := filepath.Join(dir, "requirements.yaml")
	exists, err := util.FileExists(reqfile)
	if err != nil {
		return errors.Wrapf(err, "requirements.yaml file not found in the chart directory '%s'", dir)
	}
	if exists {
		requirements, err := helm.LoadRequirementsFile(reqfile)
		if err != nil {
			return errors.Wrap(err, "failed to load the helm requirements file")
		}
		if requirements != nil {
			for _, dep := range requirements.Dependencies {
				repo := dep.Repository
				if repo != "" && !util.StringMapHasValue(installedChartRepos, repo) && repo != defaultChartRepo && !strings.HasPrefix(repo, "file:") {
					repoCounter++
					// TODO we could provide some mechanism to customise the names of repos somehow?
					err = o.addHelmBinaryRepoIfMissing(repo, "repo"+strconv.Itoa(repoCounter))
					if err != nil {
						return errors.Wrapf(err, "failed to add helm repository '%s'", repo)
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
	_, err := o.Helm().Version(false)
	if err != nil {
		return errors.Wrap(err, "failed to read the helm version")
	}
	if o.Helm().HelmBinary() == "helm" {
		return o.Helm().Init(false, "", "", true)
	} else {
		return o.Helm().Init(false, "", "", false)
	}
}

func (o *CommonOptions) helmInitDependencyBuild(dir string, chartRepos map[string]string) (string, error) {
	o.Helm().SetCWD(dir)
	err := o.Helm().RemoveRequirementsLock()
	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrapf(err, "failed to remove requirements.lock file from chat '%s'", dir)
	}

	_, err = o.Helm().Version(false)
	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrap(err, "failed to read the helm version")
	}

	if o.Helm().HelmBinary() == "helm" {
		err = o.Helm().Init(false, "", "", true)
	} else {
		err = o.Helm().Init(false, "", "", false)
	}

	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrap(err, "failed to initialize helm")
	}
	err = o.addChartRepos(dir, o.Helm().HelmBinary(), chartRepos)
	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrap(err, "failed to add chart repositories")
	}

	// TODO due to this issue: https://github.com/kubernetes/helm/issues/4230
	// lets stick with helm2 for this step
	//
	helmBinary := o.Helm().HelmBinary()
	o.Helm().SetHelmBinary("helm")
	o.Helm().SetCWD(dir)
	err = o.Helm().BuildDependency()
	if err != nil {
		return helmBinary, errors.Wrapf(err, "failed to build the dependencies of chart '%s'", dir)
	}

	o.Helm().SetHelmBinary(helmBinary)
	_, err = o.Helm().Lint()
	if err != nil {
		return helmBinary, errors.Wrapf(err, "failed to lint the chart '%s'", dir)
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

func (o *CommonOptions) ensureHelm() error {
	_, err := o.Helm().Version(false)
	if err == nil {
		return nil
	}
	err = o.installHelm()
	if err != nil {
		return errors.Wrap(err, "failed to install helm")
	}
	initOpts := InitOptions{
		CommonOptions: *o,
	}
	return initOpts.initHelm()
}
