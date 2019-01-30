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
	branchNameText := "add-app-" + app + "-" + version
	title := fmt.Sprintf("Add %s %s", app, version)
	message := fmt.Sprintf("Add app %s %s", app, version)

	options := environments.EnvironmentPullRequestOptions{
		ConfigGitFn: o.ConfigureGitFn,
		Gitter:      o.Gitter,
		ModifyChartFn: environments.CreateAddRequirementFn(app, alias, version,
			repository, o.valuesFiles, dir, o.Verbose),
		GitProvider: o.GitProvider,
	}

	pullRequestInfo, err := options.Create(o.DevEnv, &branchNameText, &title, &message, o.EnvironmentsDir, nil)

	if err != nil {
		return errors.Wrapf(err, "creating pr for %s", app)
	}
	log.Infof("Added app via Pull Request %s\n", pullRequestInfo.PullRequest.URL)
	return nil
}

// UpgradeApp upgrades the app (or all apps if empty) to a version (
// or latest if empty) from a repository with username and password.
// If one app is being upgraded an alias can be specified.
func (o *GitOpsOptions) UpgradeApp(app string, version string, repository string, username string, password string,
	alias string) error {
	var branchNameText string
	var title string
	var message string
	all := true

	if app != "" {
		all = false
		if version == "" {
			version = "latest"
		}
		branchNameText = fmt.Sprintf("upgrade-app-%s-%s", app, version)
	} else {
		branchNameText = fmt.Sprintf("upgrade-all-apps")
		title = fmt.Sprintf("Upgrade all apps")
		message = fmt.Sprintf("Upgrade all apps:\n")
	}

	upgraded := false
	modifyChartFn := func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
		templates map[string]string, dir string) error {
		for _, d := range requirements.Dependencies {
			upgrade := false
			// We need to ignore the platform
			if d.Name == "jenkins-x-platform" {
				upgrade = false
			} else if all {
				upgrade = true
			} else {
				if d.Name == app && d.Alias == alias {
					upgrade = true

				}
			}
			if upgrade {
				upgraded = true
				if all || version == "" {
					var err error
					version, err = helm.GetLatestVersion(d.Name, repository, username, password, o.Helmer)
					if err != nil {
						return err
					}
					if o.Verbose {
						log.Infof("No version specified so using latest version which is %s\n", util.ColorInfo(version))
					}
				}
				// Do the upgrade
				oldVersion := d.Version
				d.Version = version
				if !all {
					title = fmt.Sprintf("Upgrade %s to %s", app, version)
					message = fmt.Sprintf("Upgrade %s from %s to %s", app, oldVersion, version)
				} else {
					message = fmt.Sprintf("%s\n* %s from %s to %s", message, d.Name, oldVersion, version)
				}
			}
		}
		return nil
	}

	options := environments.EnvironmentPullRequestOptions{
		ConfigGitFn:   o.ConfigureGitFn,
		Gitter:        o.Gitter,
		ModifyChartFn: modifyChartFn,
		GitProvider:   o.GitProvider,
	}
	_, err := options.Create(o.DevEnv, &branchNameText, &title, &message, o.EnvironmentsDir, nil)
	if err != nil {
		return err
	}

	if !upgraded {
		log.Infof("No upgrades available\n")
	}
	return nil
}

// DeleteApp deletes the app with alias
func (o *GitOpsOptions) DeleteApp(app string, alias string) error {

	modifyChartFn := func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
		templates map[string]string, dir string) error {
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
	branchNameText := "delete-app-" + app
	title := fmt.Sprintf("Delete %s", app)
	message := fmt.Sprintf("Delete app %s", app)

	options := environments.EnvironmentPullRequestOptions{
		ConfigGitFn:   o.ConfigureGitFn,
		Gitter:        o.Gitter,
		ModifyChartFn: modifyChartFn,
		GitProvider:   o.GitProvider,
	}

	pullRequestInfo, err := options.Create(o.DevEnv, &branchNameText, &title,
		&message,
		o.EnvironmentsDir, nil)
	if err != nil {
		return err
	}
	log.Infof("Delete app via Pull Request %s\n", pullRequestInfo.PullRequest.URL)
	return nil
}
