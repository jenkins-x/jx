package auth

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/util"
	"sigs.k8s.io/yaml"
)

// fileConfigSaver is a ConfigSaver that saves its config to the local filesystem
type fileConfigLoadSaver struct {
	fileName string
}

// NewFileConfigService
func NewFileConfigService(filename string) (ConfigService, error) {
	fls, err := newFileConfigLoadSaver(filename)
	return newConfigService(fls, fls), err
}

// newFileConfigLoadSavercreates a new FileBasedAuthConfigService that stores its data under the given filename
// If the fileName is an absolute path, it will be used. If it is a simple filename, it will be stored in the default
// Config directory
func newFileConfigLoadSaver(fileName string) (*fileConfigLoadSaver, error) {
	svc := &fileConfigLoadSaver{}
	// If the fileName is an absolute path, use that. Otherwise treat it as a config filename to be used in
	if fileName == filepath.Base(fileName) {
		dir, err := util.ConfigDir()
		if err != nil {
			return svc, err
		}
		svc.fileName = filepath.Join(dir, fileName)
	} else {
		svc.fileName = fileName
	}
	return svc, nil
}

// LoadConfig loads the configuration
func (s *fileConfigLoadSaver) LoadConfig() (*Config, error) {
	config := &Config{}
	fileName := s.fileName
	if fileName != "" {
		exists, err := util.FileExists(fileName)
		if err != nil {
			return config, fmt.Errorf("Could not check if file exists %s due to %s", fileName, err)
		}
		if exists {
			data, err := ioutil.ReadFile(fileName)
			if err != nil {
				return config, fmt.Errorf("Failed to load file %s due to %s", fileName, err)
			}
			err = yaml.Unmarshal(data, config)
			if err != nil {
				return config, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
			}
		}
	}
	return config, nil
}

// SaveConfig saves the configuration to disk
func (s *fileConfigLoadSaver) SaveConfig(config *Config) error {
	fileName := s.fileName
	if fileName == "" {
		return fmt.Errorf("no filename defined")
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
}
