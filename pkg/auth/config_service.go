package auth

import (
	"fmt"
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
)

func (s *AuthConfigService) Config() *AuthConfig {
	if s.config == nil {
		s.config = &AuthConfig{}
	}
	return s.config
}

func (s *AuthConfigService) SetConfig(c *AuthConfig) {
	s.config = c
}

// LoadConfig loads the configuration from the users JX config directory
func (s *AuthConfigService) LoadConfig() (*AuthConfig, error) {
	config := s.Config()
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

// HasConfigFile returns true if we have a config file
func (s *AuthConfigService) HasConfigFile() (bool, error) {
	fileName := s.FileName
	if fileName != "" {
		exists, err := util.FileExists(fileName)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
	return false, nil
}

// SaveConfig saves the configuration to disk
func (s *AuthConfigService) SaveConfig() error {
	fileName := s.FileName
	if fileName == "" {
		return fmt.Errorf("No filename defined!")
	}
	data, err := yaml.Marshal(s.config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, DefaultWritePermissions)
}

// SaveUserAuth saves the given user auth for the server url
func (s *AuthConfigService) SaveUserAuth(url string, userAuth *UserAuth) error {
	config := s.config
	config.SetUserAuth(url, userAuth)
	user := userAuth.Username
	if user != "" {
		config.DefaultUsername = user
	}

	config.CurrentServer = url
	return s.SaveConfig()
}

// DeleteServer removes the given server from the configuration
func (s *AuthConfigService) DeleteServer(url string) error {
	s.config.DeleteServer(url)
	return s.SaveConfig()
}
