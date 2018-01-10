package auth

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	DefaultWritePermissions = 0760
)

type AuthServer struct {
	URL   string
	Users []UserAuth
	Name  string

	CurrentUser string
}

type UserAuth struct {
	Username    string
	ApiToken    string
	BearerToken string
}

type AuthConfig struct {
	Servers []AuthServer

	DefaultUsername string
	CurrentServer   string
}

func (s *AuthServer) Label() string {
	if s.Name != "" {
		return s.Name
	}
	return s.URL
}

func (s *AuthServer) Description() string {
	if s.Name != "" {
		return s.Name + " at " + s.URL
	}
	return s.URL
}

func (c *AuthConfig) FindUserAuths(serverURL string) []UserAuth {
	for _, server := range c.Servers {
		if server.URL == serverURL {
			return server.Users
		}
	}
	return []UserAuth{}
}

// FindUserAuth finds the auth for the given user name
// if no username is specified and there is only one auth then return that else nil
func (c *AuthConfig) FindUserAuth(serverURL string, username string) *UserAuth {
	auths := c.FindUserAuths(serverURL)
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
	config   AuthConfig
}

func (c *AuthConfig) SetUserAuth(url string, auth UserAuth) {
	username := auth.Username
	for i, server := range c.Servers {
		if server.URL == url {
			for j, a := range server.Users {
				if a.Username == auth.Username {
					c.Servers[i].Users[j] = auth
					c.Servers[i].CurrentUser = username
					return
				}
			}
			c.Servers[i].Users = append(c.Servers[i].Users, auth)
			c.Servers[i].CurrentUser = username
			return
		}
	}
	c.Servers = append(c.Servers, AuthServer{
		URL:         url,
		Users:       []UserAuth{auth},
		CurrentUser: username,
	})
}

func (s *AuthConfigService) Config() *AuthConfig {
	return &s.config
}

func (s *AuthConfigService) SetConfig(c AuthConfig) {
	s.config = c
}

// LoadConfig loads the configuration from the users JX config directory
func (s *AuthConfigService) LoadConfig() (*AuthConfig, error) {
	config := &s.config
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

func (c *AuthConfig) PickServer(message string) (*AuthServer, error) {
	if c.Servers == nil || len(c.Servers) == 0 {
		return nil, fmt.Errorf("No servers available!")
	}
	if len(c.Servers) == 1 {
		return &c.Servers[0], nil
	}
	urls := []string{}
	for _, s := range c.Servers {
		urls = append(urls, s.URL)
	}
	url := ""
	if len(urls) > 1 {
		prompt := &survey.Select{
			Message: message,
			Options: urls,
		}
		err := survey.AskOne(prompt, &url, nil)
		if err != nil {
			return nil, err
		}
	}
	for _, s := range c.Servers {
		if s.URL == url {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("Could not find server for URL %s", url)
}

func (c *AuthConfig) PickServerUserAuth(url string, message string) (UserAuth, error) {
	userAuths := c.FindUserAuths(url)
	if len(userAuths) == 1 {
		return userAuths[0], nil
	}
	if len(userAuths) > 1 {
		usernames := []string{}
		m := map[string]UserAuth{}
		for _, ua := range userAuths {
			name := ua.Username
			usernames = append(usernames, name)
			m[name] = ua
		}
		username := ""
		prompt := &survey.Select{
			Message: message,
			Options: usernames,
		}
		err := survey.AskOne(prompt, &username, nil)
		if err != nil {
			return UserAuth{}, err
		}
		return m[username], nil
	}
	return UserAuth{}, nil
}

// EditUserAuth Lets the user input/edit the user auth
func (config *AuthConfig) EditUserAuth(auth *UserAuth, defaultUserName string) error {
	// default the user name if its empty
	defaultUsername := config.DefaultUsername
	if defaultUsername == "" {
		defaultUsername = defaultUserName
	}
	if auth.Username == "" {
		auth.Username = defaultUsername
	}

	var qs = []*survey.Question{
		{
			Name: "username",
			Prompt: &survey.Input{
				Message: "User name:",
				Default: auth.Username,
			},
			Validate: survey.Required,
		},
		{
			Name: "apiToken",
			Prompt: &survey.Input{
				Message: "API Token:",
				Default: auth.ApiToken,
			},
			Validate: survey.Required,
		},
	}
	return survey.Ask(qs, auth)
}

// SaveUserAuth saves the given user auth for the server url
func (s *AuthConfigService) SaveUserAuth(url string, userAuth *UserAuth) error {
	config := s.config
	config.SetUserAuth(url, *userAuth)
	user := userAuth.Username
	if user != "" {
		config.DefaultUsername = user
	}
	config.CurrentServer = url
	return s.SaveConfig()
}
