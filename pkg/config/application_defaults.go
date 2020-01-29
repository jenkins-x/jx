package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

const (
	// ApplicationsDefaultsConfigFileName is the name of the applications configuration file
	ApplicationsDefaultsConfigFileName = "defaults.yml"
)

// ApplicationConfig contains applications to install during boot
type ApplicationDefaultsConfig struct {
	// Namespace the default namespace to install this app into
	Namespace string `json:"namespace,omitempty"`
	// Phase the boot phase this app should be installed in. Leave blank if you are not sure.
	// things like ingress controllers are in 'system' and most other things default to the 'apps' phase
	Phase string `json:"phase,omitempty"`
}

// LoadApplicationDefaultsConfig loads the boot application default configuration
// used to default values if the user does not specify any configuration when doing `jx add app foo`
// to try encourage default configurations across installations (e.g. using default namespace names and so forth)
func LoadApplicationDefaultsConfig(dir string) (*ApplicationDefaultsConfig, string, error) {
	fileName := ApplicationsDefaultsConfigFileName
	if dir != "" {
		fileName = filepath.Join(dir, fileName)
	}

	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, fileName, errors.Errorf("error looking up %s in directory %s", fileName, dir)
	}

	config := &ApplicationDefaultsConfig{}
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
	return config, fileName, err
}
