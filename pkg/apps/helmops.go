package apps

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// HelmOpsOptions is the options used for Helm Operations for apps
type HelmOpsOptions struct {
	*InstallOptions
}

// AddApp adds the app with a version and releaseName from the chart from the repository with username and password.
// A values file or a slice of name=value pairs can be passed in to configure the chart
func (o *HelmOpsOptions) AddApp(app string, chart string, name string, version string, values []byte, repository string,
	username string, password string, releaseName string, setValues []string, helmUpdate bool) error {

	parsedSetValues := make([]string, 0)
	for _, vs := range setValues {
		parsedSetValues = append(parsedSetValues, strings.Split(vs, ",")...)
	}

	err := helm.InstallFromChartOptions(helm.InstallChartOptions{
		ReleaseName: releaseName,
		Chart:       chart,
		Version:     version,
		Ns:          o.Namespace,
		HelmUpdate:  helmUpdate,
		SetValues:   parsedSetValues,
		ValueFiles:  o.valuesFiles.Items,
		Repository:  repository,
		Username:    username,
		Password:    password,
	}, o.Helmer, o.KubeClient, o.InstallTimeout, o.VaultClient)
	if err != nil {
		return errors.Wrapf(err, "failed to install app %s", app)
	}
	// Attach the current values.yaml
	appCRDName := fmt.Sprintf("%s-%s", releaseName, name)

	err = StashValues(values, appCRDName, o.JxClient, o.Namespace, chart, repository)
	if err != nil {
		return errors.Wrapf(err, "attaching values.yaml to %s", appCRDName)
	}
	log.Infof("Successfully installed %s %s\n", util.ColorInfo(name), util.ColorInfo(version))
	return nil
}

//DeleteApp deletes the app, optionally allowing the user to set the releaseName
func (o *HelmOpsOptions) DeleteApp(app string, releaseName string, purge bool) error {
	if releaseName == "" {
		releaseName = app
	}
	return o.Helmer.DeleteRelease(o.Namespace, releaseName, purge)
}

//UpgradeApp upgrades the app with releaseName (or all apps if the app name is empty) to the specified version (
// or the latest version if the version is empty) using the repository with username and password
func (o *HelmOpsOptions) UpgradeApp(app string, version string, repository string, username string, password string,
	releaseName string, alias string, helmUpdate bool) error {

	// TODO implement this!
	/*var branchNameText string
	var title string
	var message string
	all := true

	if app != "" {
		all = false
		if version == "" {
			version = "latest"
		}
		branchNameText = fmt.Sprintf("upgrade-app-%s-%s", app, version)
	}

	branchNameText = fmt.Sprintf("upgrade-all-apps")
	title = fmt.Sprintf("Upgrade all apps")
	message = fmt.Sprintf("Upgrade all apps:\n")

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

	if !upgraded {
		log.Infof("No upgrades available\n")
	}*/
	return nil
}
