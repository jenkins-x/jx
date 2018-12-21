package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/kube/services"

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
	u, err := services.FindServiceURL(client, ns, kube.ServiceChartMuseum)
	if err != nil {
		return errors.Wrapf(err, "failed to find the service URL of the ChartMuseum")
	}
	u2, err := url.Parse(u)
	if err != nil {
		return errors.Wrap(err, "failed to parse the ChartMuseum URL")
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
		log.Infof("Adding missing Helm repo: %s %s\n", util.ColorInfo(repoName), util.ColorInfo(helmUrl))
		err = o.retry(6, 10*time.Second, func() (err error) {
			err = o.Helm().AddRepo(repoName, helmUrl)
			if err == nil {
				log.Infof("Successfully added Helm repository %s.\n", repoName)
			}
			return errors.Wrapf(err, "failed to add the repository '%s' with URL '%s'", repoName, helmUrl)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// installChart installs the given chart
func (o *CommonOptions) installChart(releaseName string, chart string, version string, ns string, helmUpdate bool,
	setValues []string, valueFiles []string, repo string) error {
	return o.installChartOptions(helm.InstallChartOptions{ReleaseName: releaseName, Chart: chart, Version: version,
		Ns: ns, HelmUpdate: helmUpdate, SetValues: setValues, ValueFiles: valueFiles, Repository: repo})
}

// installChartAt installs the given chart
func (o *CommonOptions) installChartAt(dir string, releaseName string, chart string, version string, ns string,
	helmUpdate bool, setValues []string, valueFiles []string, repo string) error {
	return o.installChartOptions(helm.InstallChartOptions{Dir: dir, ReleaseName: releaseName, Chart: chart,
		Version: version, Ns: ns, HelmUpdate: helmUpdate, SetValues: setValues, ValueFiles: valueFiles, Repository: repo})
}

func (o *CommonOptions) installChartOptions(options helm.InstallChartOptions) error {
	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	return helm.InstallFromChartOptions(options, o.Helm(), client, defaultInstallTimeout)
}

// deleteChart deletes the given chart
func (o *CommonOptions) deleteChart(releaseName string, purge bool) error {
	_, ns, err := o.KubeClient()
	if err != nil {
		return err
	}

	return o.Helm().DeleteRelease(ns, releaseName, purge)
}

func (o *CommonOptions) FindHelmChart() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get the current working directory")
	}
	o.Helm().SetCWD(dir)
	return o.Helm().FindChart()
}

func (o *CommonOptions) DiscoverAppName() (string, error) {
	answer := ""
	chartFile, err := o.FindHelmChart()
	if err != nil {
		return answer, err
	}
	if chartFile != "" {
		return helm.LoadChartName(chartFile)
	}

	gitInfo, err := o.Git().Info("")
	if err != nil {
		return answer, err
	}

	if gitInfo == nil {
		return answer, fmt.Errorf("no git info found to discover app name from")
	}
	answer = gitInfo.Name

	if answer == "" {
	}
	return answer, nil
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
					return errors.Wrapf(err, "failed to add the Helm repository with name '%s' and URL '%s'", name, url)
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
			return errors.Wrap(err, "failed to load the Helm requirements file")
		}
		if requirements != nil {
			for _, dep := range requirements.Dependencies {
				repo := dep.Repository
				if repo != "" && !util.StringMapHasValue(installedChartRepos, repo) && repo != defaultChartRepo && !strings.HasPrefix(repo, "file:") && !strings.HasPrefix(repo, "alias:") {
					repoCounter++
					// TODO we could provide some mechanism to customise the names of repos somehow?
					err = o.addHelmBinaryRepoIfMissing(repo, "repo"+strconv.Itoa(repoCounter))
					if err != nil {
						return errors.Wrapf(err, "failed to add Helm repository '%s'", repo)
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
	if o.Helm().HelmBinary() == "helm" {
		// need to check the tiller settings at this point
		_, noTiller, helmTemplate, err := o.TeamHelmBin()
		if err != nil {
			return errors.Wrap(err, "failed to access team settings")
		}

		if noTiller || helmTemplate {
			return o.Helm().Init(true, "", "", false)
		} else {
			return o.Helm().Init(false, "", "", true)
		}
	} else {
		return o.Helm().Init(false, "", "", false)
	}
}

func (o *CommonOptions) helmInitDependency(dir string, chartRepos map[string]string) (string, error) {
	o.Helm().SetCWD(dir)
	err := o.Helm().RemoveRequirementsLock()
	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrapf(err, "failed to remove requirements.lock file from chat '%s'", dir)
	}

	if o.Helm().HelmBinary() == "helm" {
		// need to check the tiller settings at this point
		_, noTiller, helmTemplate, err := o.TeamHelmBin()
		if err != nil {
			return o.Helm().HelmBinary(),
				errors.Wrap(err, "failed to access team settings")
		}

		if noTiller || helmTemplate {
			err = o.Helm().Init(true, "", "", false)
		} else {
			err = o.Helm().Init(false, "", "", true)
		}
	} else {
		err = o.Helm().Init(false, "", "", false)
	}

	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrap(err, "failed to initialize Helm")
	}
	err = o.addChartRepos(dir, o.Helm().HelmBinary(), chartRepos)
	if err != nil {
		return o.Helm().HelmBinary(),
			errors.Wrap(err, "failed to add chart repositories")
	}

	return o.Helm().HelmBinary(), nil
}

func (o *CommonOptions) helmInitDependencyBuild(dir string, chartRepos map[string]string) (string, error) {
	helmBin, err := o.helmInitDependency(dir, chartRepos)
	if err != nil {
		return helmBin, err
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

func (o *CommonOptions) helmInitRecursiveDependencyBuild(dir string, chartRepos map[string]string) error {
	_, err := o.helmInitDependency(dir, chartRepos)
	if err != nil {
		return errors.Wrap(err, "initializing Helm")
	}

	helmBinary := o.Helm().HelmBinary()
	o.Helm().SetHelmBinary("helm")
	o.Helm().SetCWD(dir)
	err = o.Helm().BuildDependency()
	if err != nil {
		return errors.Wrapf(err, "failed to build the dependencies of chart '%s'", dir)
	}

	reqFilePath := filepath.Join(dir, "requirements.yaml")
	reqs, err := helm.LoadRequirementsFile(reqFilePath)
	if err != nil {
		return errors.Wrap(err, "loading the requirements file")
	}

	type chartDep struct {
		path string
		deps []*helm.Dependency
	}

	baseChartPath := filepath.Join(dir, "charts")
	depQueue := []chartDep{{
		path: baseChartPath,
		deps: reqs.Dependencies,
	}}

	for {
		if len(depQueue) == 0 {
			break
		}
		currChartDep := depQueue[0]
		depQueue = depQueue[1:]
		for _, dep := range currChartDep.deps {
			chartArchive := filepath.Join(currChartDep.path, fmt.Sprintf("%s-%s.tgz", dep.Name, dep.Version))
			chartPath := filepath.Join(currChartDep.path, dep.Name)
			err := os.MkdirAll(chartPath, os.ModePerm)
			if err != nil {
				return errors.Wrap(err, "creating directory")
			}
			err = util.UnTargz(chartArchive, chartPath, []string{})
			if err != nil {
				return errors.Wrap(err, "extracting chart")
			}
			o.Helm().SetCWD(chartPath)
			err = o.Helm().BuildDependency()
			if err != nil {
				return errors.Wrap(err, "building Helm dependency")
			}
			chartReqFile := filepath.Join(chartPath, "requirements.yaml")
			reqs, err := helm.LoadRequirementsFile(chartReqFile)
			if err != nil {
				return errors.Wrap(err, "loading the requirements file")
			}
			if len(reqs.Dependencies) > 0 {
				depQueue = append(depQueue, chartDep{
					path: filepath.Join(chartPath, "charts"),
					deps: reqs.Dependencies,
				})
			}
		}
	}

	o.Helm().SetHelmBinary(helmBinary)
	_, err = o.Helm().Lint()
	if err != nil {
		return errors.Wrapf(err, "linting the chart '%s'", dir)
	}

	return nil
}

func (o *CommonOptions) defaultReleaseCharts() map[string]string {
	releasesURL := o.releaseChartMuseumUrl()
	answer := map[string]string{
		"jenkins-x": DEFAULT_CHARTMUSEUM_URL,
	}
	if releasesURL != "" {
		answer["releases"] = releasesURL
	}
	return answer
}

func (o *CommonOptions) releaseChartMuseumUrl() string {
	chartRepo := os.Getenv("CHART_REPOSITORY")
	if chartRepo == "" {
		if o.Factory.IsInCDPipeline() {
			chartRepo = defaultChartRepo
			log.Warnf("No $CHART_REPOSITORY defined so using the default value of: %s\n", defaultChartRepo)
		} else {
			return ""
		}
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
		return errors.Wrap(err, "failed to install Helm")
	}
	initOpts := InitOptions{
		CommonOptions: *o,
	}
	return initOpts.initHelm()
}
