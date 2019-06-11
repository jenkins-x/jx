package auth

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/util"
	"sigs.k8s.io/yaml"
)

// fileAuthConfigSaver is a ConfigSaver that saves its config to the local filesystem
type fileAuthConfigSaver struct {
	FileName string
}

// NewFileAuthConfigService
func NewFileAuthConfigService(filename string) (ConfigService, error) {
	saver, err := newFileAuthSaver(filename)
	return NewAuthConfigService(saver), err
}

// newFileBasedAuthConfigSaver creates a new FileBasedAuthConfigService that stores its data under the given filename
// If the fileName is an absolute path, it will be used. If it is a simple filename, it will be stored in the default
// Config directory
func newFileAuthSaver(fileName string) (ConfigSaver, error) {
	svc := &fileAuthConfigSaver{}
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
func (s *fileAuthConfigSaver) LoadConfig() (*AuthConfig, error) {
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
func (s *fileAuthConfigSaver) SaveConfig(config *AuthConfig) error {
	fileName := s.FileName
	if fileName == "" {
		return fmt.Errorf("no filename defined")
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
}
