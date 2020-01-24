package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	// ApplicationsConfigFileName is the name of the applications configuration file
	ApplicationsConfigFileName = "jx-apps.yml"
)

// ApplicationConfig contains applications to install during boot
type ApplicationConfig struct {
	// Applications of applications
	Applications []Application `json:"applications"`
	// DefaultNamespace the default namespace to install applications into
	DefaultNamespace string `json:"defaultNamespace"`
}

// Application is an application to install during boot
type Application struct {
	// Name of the application / helm chart
	Name string `json:"name"`
	// Repository the helm repository
	Repository string `json:"repository"`
	// Namespace to install the application into
	Namespace string `json:"namespace"`
}

// LoadApplicationsConfig loads the boot applications configuration file
// if there is not a file called `jx-apps.yml` in the given dir we will scan up the parent
// directories looking for the requirements file as we often run 'jx' steps in sub directories.
func LoadApplicationsConfig(dir string) (*ApplicationConfig, error) {
	fileName := ApplicationsConfigFileName
	if dir != "" {
		fileName = filepath.Join(dir, fileName)
	}

	exists, err := util.FileExists(fileName)
	if err != nil || !exists {
		return nil, errors.Errorf("no %s found in directory %s", fileName, dir)
	}

	config := &ApplicationConfig{}

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return config, fmt.Errorf("Failed to load file %s due to %s", fileName, err)
	}
	validationErrors, err := util.ValidateYaml(config, data)
	if err != nil {
		return config, fmt.Errorf("failed to validate YAML file %s due to %s", fileName, err)
	}
	if len(validationErrors) > 0 {
		return config, fmt.Errorf("Validation failures in YAML file %s:\n%s", fileName, strings.Join(validationErrors, "\n"))
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return config, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
	}

	return config, err
}
