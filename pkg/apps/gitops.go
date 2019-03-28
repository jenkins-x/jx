package apps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/jenkins-x/jx/pkg/environments"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
)

// GitOpsOptions is the options used for Git Operations for apps
type GitOpsOptions struct {
	*InstallOptions
}

// AddApp adds the app with version rooted in dir from the repository. An alias can be specified.
func (o *GitOpsOptions) AddApp(app string, dir string, version string, repository string, alias string) error {
	details := environments.PullRequestDetails{
		BranchName: "add-app-" + app + "-" + version,
		Title:      fmt.Sprintf("Add %s %s", app, version),
		Message:    fmt.Sprintf("Add app %s %s", app, version),
	}

	options := environments.EnvironmentPullRequestOptions{
		ConfigGitFn: o.ConfigureGitFn,
		Gitter:      o.Gitter,
		ModifyChartFn: environments.CreateAddRequirementFn(app, alias, version,
			repository, o.valuesFiles, dir, o.Verbose, o.Helmer),
		GitProvider: o.GitProvider,
	}

	info, err := options.Create(o.DevEnv, o.EnvironmentsDir, &details, nil)

	if err != nil {
		return errors.Wrapf(err, "creating pr for %s", app)
	}
	log.Infof("Added app via Pull Request %s\n", info.PullRequest.URL)
	return nil
}

// UpgradeApp upgrades the app (or all apps if empty) to a version (
// or latest if empty) from a repository with username and password.
// If one app is being upgraded an alias can be specified.
func (o *GitOpsOptions) UpgradeApp(app string, version string, repository string, username string, password string,
	alias string, interrogateChartFunc func(dir string, existing map[string]interface{}) (*ChartDetails,
		error)) error {
	all := true
	details := environments.PullRequestDetails{}

	if app != "" {
		all = false
		if version == "" {
			version = "latest"
		}
		details.BranchName = fmt.Sprintf("upgrade-app-%s-%s", app, version)
	} else {
		details.BranchName = fmt.Sprintf("upgrade-all-apps")
		details.Title = fmt.Sprintf("Upgrade all apps")
		details.Message = fmt.Sprintf("Upgrade all apps:\n")
	}

	var interrogateCleanup func()
	defer func() {
		if interrogateCleanup != nil {
			interrogateCleanup()
		}
	}()
	inspectChartFunc := func(chartDir string, values map[string]interface{}) error {
		chartDetails, err := interrogateChartFunc(chartDir, values)
		interrogateCleanup = chartDetails.Cleanup
		if err != nil {
			return errors.Wrapf(err, "asking questions for %s", chartDir)
		}
		return nil
	}

	options := environments.EnvironmentPullRequestOptions{
		ConfigGitFn: o.ConfigureGitFn,
		Gitter:      o.Gitter,
		ModifyChartFn: environments.CreateUpgradeRequirementsFn(all, app, alias, version, username, password,
			o.Helmer, inspectChartFunc, o.Verbose, o.valuesFiles),
		GitProvider: o.GitProvider,
	}
	_, err := options.Create(o.DevEnv, o.EnvironmentsDir, &details, nil)
	if err != nil {
		return err
	}
	return nil
}

// DeleteApp deletes the app with alias
func (o *GitOpsOptions) DeleteApp(app string, alias string) error {

	modifyChartFn := func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
		templates map[string]string, dir string, details *environments.PullRequestDetails) error {
		// See if the app already exists in requirements
		found := false
		for i, d := range requirements.Dependencies {
			if d.Name == app && d.Alias == alias {
				found = true
				requirements.Dependencies[i] = nil
			}
		}
		// If app not found, add it
		if !found {
			a := app
			if alias != "" {
				a = fmt.Sprintf("%s with alias %s", a, alias)
			}
			return fmt.Errorf("unable to delete app %s as not installed", app)
		}
		if info, err := os.Stat(filepath.Join(dir, app)); err == nil {
			if info.IsDir() {
				err := util.DeleteFile(info.Name())
				if err != nil {
					return err
				}
			} else {
				log.Warnf("Not removing %s for %s because it is not a directory", info.Name(), app)
			}
		}
		return nil
	}
	details := environments.PullRequestDetails{
		BranchName: "delete-app-" + app,
		Title:      fmt.Sprintf("Delete %s", app),
		Message:    fmt.Sprintf("Delete app %s", app),
	}

	options := environments.EnvironmentPullRequestOptions{
		ConfigGitFn:   o.ConfigureGitFn,
		Gitter:        o.Gitter,
		ModifyChartFn: modifyChartFn,
		GitProvider:   o.GitProvider,
	}

	info, err := options.Create(o.DevEnv, o.EnvironmentsDir, &details, nil)
	if err != nil {
		return err
	}
	log.Infof("Delete app via Pull Request %s\n", info.PullRequest.URL)
	return nil
}
