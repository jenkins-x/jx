package auth

import (
	"fmt"
	"sort"

	"gopkg.in/AlecAivazis/survey.v1"
)

func (c *AuthConfig) FindUserAuths(serverURL string) []*UserAuth {
	for _, server := range c.Servers {
		if server.URL == serverURL {
			return server.Users
		}
	}
	return []*UserAuth{}
}

func (c *AuthConfig) GetOrCreateUserAuth(url string, username string) *UserAuth {
	user := c.FindUserAuth(url, username)
	if user != nil {
		return user
	}
	server := c.GetOrCreateServer(url)
	if server.Users == nil {
		server.Users = []*UserAuth{}
	}
	user = &UserAuth{
		Username: username,
	}
	server.Users = append(server.Users, user)
	for _, user := range server.Users {
		if user.Username == username {
			return user
		}
	}
	return nil
}

// FindUserAuth finds the auth for the given user name
// if no username is specified and there is only one auth then return that else nil
func (c *AuthConfig) FindUserAuth(serverURL string, username string) *UserAuth {
	auths := c.FindUserAuths(serverURL)
	if username == "" {
		if len(auths) == 1 {
			return auths[0]
		} else {
			return nil
		}
	}
	for _, auth := range auths {
		if auth.Username == username {
			return auth
		}
	}
	return nil
}

func (config *AuthConfig) IndexOfServerName(name string) int {
	for i, server := range config.Servers {
		if server.Name == name {
			return i
		}
	}
	return -1
}

func (c *AuthConfig) SetUserAuth(url string, auth *UserAuth) {
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
	c.Servers = append(c.Servers, &AuthServer{
		URL:         url,
		Users:       []*UserAuth{auth},
		CurrentUser: username,
	})
}

func (c *AuthConfig) GetServer(url string) *AuthServer {
	for _, s := range c.Servers {
		if s.URL == url {
			return s
		}
	}
	return nil
}

func (c *AuthConfig) GetServerByName(name string) *AuthServer {
	for _, s := range c.Servers {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func (c *AuthConfig) GetOrCreateServer(url string) *AuthServer {
	name := ""
	kind := ""
	if url == "github.com" {
		name = "GitHub"
		kind = "github"
	}
	return c.GetOrCreateServerName(url, name, kind)
}

func (c *AuthConfig) GetOrCreateServerName(url string, name string, kind string) *AuthServer {
	s := c.GetServer(url)
	if s == nil {
		if c.Servers == nil {
			c.Servers = []*AuthServer{}
		}
		s = &AuthServer{
			URL:   url,
			Users: []*UserAuth{},
			Name:  name,
			Kind:  kind,
		}
		c.Servers = append(c.Servers, s)
	}
	return s
}

func (c *AuthConfig) PickServer(message string) (*AuthServer, error) {
	if c.Servers == nil || len(c.Servers) == 0 {
		return nil, fmt.Errorf("No servers available!")
	}
	if len(c.Servers) == 1 {
		return c.Servers[0], nil
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
			return s, nil
		}
	}
	return nil, fmt.Errorf("Could not find server for URL %s", url)
}

func (c *AuthConfig) PickServerUserAuth(server *AuthServer, message string, batchMode bool) (*UserAuth, error) {
	url := server.URL
	userAuths := c.FindUserAuths(url)
	if len(userAuths) == 1 {
		auth := userAuths[0]
		if batchMode {
			return auth, nil
		}
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("Do you wish to use %s as the %s", auth.Username, message),
			Default: true,
		}
		flag := false
		err := survey.AskOne(confirm, &flag, nil)
		if err != nil {
			return auth, err
		}
		if flag {
			return auth, nil
		}

		// lets create a new user name
		prompt := &survey.Input{
			Message: message,
		}
		username := ""
		err = survey.AskOne(prompt, &username, nil)
		if err != nil {
			return auth, err
		}
		return c.GetOrCreateUserAuth(url, username), nil
	}
	if len(userAuths) > 1 {
		usernames := []string{}
		m := map[string]*UserAuth{}
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
			return &UserAuth{}, err
		}
		return m[username], nil
	}
	return &UserAuth{}, nil
}

// EditUserAuth Lets the user input/edit the user auth
func (config *AuthConfig) EditUserAuth(serverLabel string, auth *UserAuth, defaultUserName string, editUser, batchMode bool) error {
	// default the user name if its empty
	defaultUsername := config.DefaultUsername
	if defaultUsername == "" {
		defaultUsername = defaultUserName
	}
	if auth.Username == "" {
		auth.Username = defaultUsername
	}

	if batchMode {
		if auth.Username == "" {
			fmt.Errorf("Running in batch mode and no default git username found")
		}
		if auth.ApiToken == "" {
			fmt.Errorf("Running in batch mode and no default api token found")
		}
		return nil
	}
	var qs = []*survey.Question{}

	if editUser || auth.Username == "" {
		qs = append(qs, &survey.Question{
			Name: "username",
			Prompt: &survey.Input{
				Message: serverLabel + " user name:",
				Default: auth.Username,
			},
			Validate: survey.Required,
		})
	}
	qs = append(qs, &survey.Question{
		Name: "apiToken",
		Prompt: &survey.Password{
			Message: "API Token:",
		},
		Validate: survey.Required,
	})
	return survey.Ask(qs, auth)
}

func (config *AuthConfig) GetServerNames() []string {
	answer := []string{}
	for _, server := range config.Servers {
		name := server.Name
		if name != "" {
			answer = append(answer, name)
		}
	}
	sort.Strings(answer)
	return answer
}

func (config *AuthConfig) GetServerURLs() []string {
	answer := []string{}
	for _, server := range config.Servers {
		u := server.URL
		if u != "" {
			answer = append(answer, u)
		}
	}
	sort.Strings(answer)
	return answer
}
