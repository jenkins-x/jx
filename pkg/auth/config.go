package auth

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

const (
	DefaultWritePermissions = 0760
)

type AuthServer struct {
	URL   string
	Users []UserAuth
}

type UserAuth struct {
	Username    string
	ApiToken    string
	BearerToken string
}

type AuthConfig struct {
	Servers []AuthServer

	DefaultUsername string
}

func (c *AuthConfig) FindAuths(serverURL string) []UserAuth {
	for _, server := range c.Servers {
		if server.URL == serverURL {
			return server.Users
		}
	}
	return []UserAuth{}
}

// FindAuth finds the auth for the given user name
// if no username is specified and there is only one auth then return that else nil
func (c *AuthConfig) FindAuth(serverURL string, username string) *UserAuth {
	auths := c.FindAuths(serverURL)
	if username == "" {
		if len(auths) == 1 {
			return &auths[0]
		} else {
			return nil
		}
	}
	for _, auth := range auths {
		if auth.Username == username {
			return &auth
		}
	}
	return nil
}

type AuthConfigService struct {
	FileName string
}

func (c *AuthConfig) SetAuth(url string, auth UserAuth) {
	for i, server := range c.Servers {
		if server.URL == url {
			for j, a := range server.Users {
				if a.Username == auth.Username {
					c.Servers[i].Users[j] = auth
					return
				}
			}
			c.Servers[i].Users = append(c.Servers[i].Users, auth)
			return
		}
	}
	c.Servers = append(c.Servers, AuthServer{
		URL:   url,
		Users: []UserAuth{auth},
	})
}

// LoadConfig loads the configuration from the users JX config directory
func (s *AuthConfigService) LoadConfig() (AuthConfig, error) {
	config := AuthConfig{}

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
			err = yaml.Unmarshal(data, &config)
			if err != nil {
				return config, fmt.Errorf("Failed to unmarshal YAML file %s due to %s", fileName, err)
			}
		}
	}
	return config, nil
}

// SaveConfig loads the configuration from the users JX config directory
func (s *AuthConfigService) SaveConfig(config *AuthConfig) error {
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

func CreateAuthUserFromEnvironment(prefix string) UserAuth {
	return UserAuth{
		Username:    os.Getenv(prefix + "_USERNAME"),
		ApiToken:    os.Getenv(prefix + "_API_TOKEN"),
		BearerToken: os.Getenv(prefix + "_BEARER_TOKEN"),
	}
}

func (a *UserAuth) IsInvalid() bool {
	return a.BearerToken == "" && (a.ApiToken == "" || a.Username == "")
}
