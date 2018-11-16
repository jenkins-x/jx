package auth

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

// NewFileBasedAuthConfigService
func NewFileBasedAuthConfigService(filename string) (AuthConfigService, error) {
	saver, err := newFileBasedAuthSaver(filename)
	service := GenericAuthConfigService{
		saver: saver,
	}
	return &service, err
}

// newFileBasedAuthConfigSaver creates a new FileBasedAuthConfigService that stores its data under the given filename
func newFileBasedAuthSaver(fileName string) (AuthConfigSaver, error) {
	svc := &FileBasedAuthConfigSaver{}
	// If the fileName is an absolute path, use that. Otherwise treat it as a config filename to be used in
	if fileName == filepath.Base(fileName) {
		dir, err := util.ConfigDir()
		if err != nil {
			return svc, err
		}
		svc.FileName = filepath.Join(dir, fileName)
	} else {
		svc.FileName = fileName
	}
	return svc, nil
}

// LoadConfig loads the configuration from the users JX config directory
func (s *FileBasedAuthConfigSaver) LoadConfig() (*AuthConfig, error) {
	config := &AuthConfig{}
	fileName := s.FileName
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
func (s *FileBasedAuthConfigSaver) SaveConfig(config *AuthConfig) error {
	fileName := s.FileName
	if fileName == "" {
		return fmt.Errorf("No filename defined!")
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, DefaultWritePermissions)
}
