package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/helmfile"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

const (
	// AppConfigFileName is the name of the applications configuration file
	AppConfigFileName = "jx-apps.yml"
	// PhaseSystem is installed before the apps phase
	PhaseSystem Phase = "system"
	// PhaseApps is installed after the system phase
	PhaseApps Phase = "apps"
)

// PhaseValues the string values for Phases
var PhaseValues = []string{"system", "apps"}

// AppConfig contains the apps to install during boot for helmfile / helm 3
type AppConfig struct {
	// Apps of applications
	Apps []App `json:"apps"`
	// Repositories list of helm repositories
	Repositories []helmfile.RepositorySpec `json:"repositories,omitempty"`
	// DefaultNamespace the default namespace to install applications into
	DefaultNamespace string `json:"defaultNamespace,omitempty"`
}

// App is the configuration of an app used during boot for helmfile / helm 3
type App struct {
	// Name of the application / helm chart
	Name string `json:"name,omitempty"`
	// Repository the helm repository
	Repository string `json:"repository,omitempty"`
	// Namespace to install the application into
	Namespace string `json:"namespace,omitempty"`
	// Phase of the pipeline to install application
	Phase Phase `json:"phase,omitempty"`
	// Version the version to install if you want to override the version from the Version Stream.
	// Note we recommend using the version stream for app versions
	Version string `json:"version,omitempty"`
	// Description an optional description of the app
	Description string `json:"description,omitempty"`
	// ValueFiles any explicit value files to be used
	ValueFiles []string `json:"valueFiles,omitempty"`
}

// Phase of the pipeline to install application
type Phase string

// LoadAppConfig loads the boot applications configuration file
// if there is not a file called `jx-apps.yml` in the given dir we will scan up the parent
// directories looking for the requirements file as we often run 'jx' steps in sub directories.
func LoadAppConfig(dir string) (*AppConfig, string, error) {
	fileName := AppConfigFileName
	if dir != "" {
		fileName = filepath.Join(dir, fileName)
	}

	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, fileName, errors.Errorf("error looking up %s in directory %s", fileName, dir)
	}

	config := &AppConfig{}
	if !exists {
		return config, "", nil
	}

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return config, fileName, fmt.Errorf("Failed to load file %s due to %s", fileName, err)
	}
	validationErrors, err := util.ValidateYaml(config, data)
	if err != nil {
		return config, fileName, fmt.Errorf("failed to validate YAML file %s due to %s", fileName, err)
	}
	if len(validationErrors) > 0 {
		return config, fileName, fmt.Errorf("Validation failures in YAML file %s:\n%s", fileName, strings.Join(validationErrors, "\n"))
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return config, fileName, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
	}

	// validate all phases are known types, default to apps if not specified
	for _, app := range config.Apps {
		if app.Phase != "" {
			if app.Phase != PhaseSystem && app.Phase != PhaseApps {
				return config, fileName, fmt.Errorf("failed to validate YAML file, invalid phase '%s', needed on of %v",
					string(app.Phase), PhaseValues)
			}
		}
	}

	return config, fileName, err
}

// SaveConfig saves the configuration file to the given project directory
func (c *AppConfig) SaveConfig(fileName string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}
	return nil
}
