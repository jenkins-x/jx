package apps

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

type HelmOpsOptions struct {
	*InstallOptions
}

func (o *HelmOpsOptions) AddApp(name string, chart string, version string, values []byte, repository string,
	username string, password string, setValues []string, helmUpdate bool) error {

	parsedSetValues := make([]string, 0)
	for _, vs := range setValues {
		parsedSetValues = append(parsedSetValues, strings.Split(vs, ",")...)
	}

	err := helm.InstallFromChartOptions(helm.InstallChartOptions{
		ReleaseName: name,
		Chart:       chart,
		Version:     version,
		Ns:          o.Namespace,
		HelmUpdate:  helmUpdate,
		SetValues:   parsedSetValues,
		ValueFiles:  o.valuesFiles,
		Repository:  repository,
		Username:    username,
		Password:    password,
	}, o.Helmer, o.KubeClient, o.InstallTimeout)
	if err != nil {
		return fmt.Errorf("failed to install name %s: %v", name, err)
	}
	// Attach the current values.yaml
	appCRDName := fmt.Sprintf("%s-%s", name, name)

	err = StashValues(values, appCRDName, o.JxClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "attaching values.yaml to %s", appCRDName)
	}
	log.Infof("Successfully installed %s %s\n", util.ColorInfo(name), util.ColorInfo(version))
	return nil
}

func (o *HelmOpsOptions) DeleteApp(name string, releaseName string, purge bool) error {
	if releaseName == "" {
		releaseName = name
	}
	return o.Helmer.DeleteRelease(o.Namespace, releaseName, purge)
}

func (o *HelmOpsOptions) UpgradeApp(app string, version string, repository string, username string, password string,
	alias string, helmUpdate bool) error {

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
