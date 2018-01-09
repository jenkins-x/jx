package jenkins

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

type JenkinsServer struct {
	URL   string
	Auths []JenkinsAuth
}

type JenkinsAuth struct {
	Username    string
	ApiToken    string
	BearerToken string
}

type JenkinsConfig struct {
	Servers []JenkinsServer

	DefaultUsername string
}

func (c *JenkinsConfig) FindAuths(serverURL string) []JenkinsAuth {
	for _, server := range c.Servers {
		if server.URL == serverURL {
			return server.Auths
		}
	}
	return []JenkinsAuth{}
}

// FindAuth finds the auth for the given user name
// if no username is specified and there is only one auth then return that else nil
func (c *JenkinsConfig) FindAuth(serverURL string, username string) *JenkinsAuth {
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

type JenkinsConfigService struct {
	FileName string
}

func (c *JenkinsConfig) SetAuth(url string, auth JenkinsAuth) {
	for i, server := range c.Servers {
		if server.URL == url {
			for j, a := range server.Auths {
				if a.Username == auth.Username {
					c.Servers[i].Auths[j] = auth
					return
				}
			}
			c.Servers[i].Auths = append(c.Servers[i].Auths, auth)
			return
		}
	}
	c.Servers = append(c.Servers, JenkinsServer{
		URL:   url,
		Auths: []JenkinsAuth{auth},
	})
}

// LoadConfig loads the configuration from the users JX config directory
func (s *JenkinsConfigService) LoadConfig() (JenkinsConfig, error) {
	config := JenkinsConfig{}

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
func (s *JenkinsConfigService) SaveConfig(config *JenkinsConfig) error {
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

func CreateJenkinsAuthFromEnvironment() JenkinsAuth {
	return JenkinsAuth{
		Username:    os.Getenv("JENKINS_USERNAME"),
		ApiToken:    os.Getenv("JENKINS_API_TOKEN"),
		BearerToken: os.Getenv("JENKINS_BEARER_TOKEN"),
	}
}

func (a *JenkinsAuth) IsInvalid() bool {
	return a.BearerToken == "" && (a.ApiToken == "" || a.Username == "")
}
