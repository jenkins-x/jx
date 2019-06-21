package auth

import (
	"sort"
	"strings"

	"github.com/pkg/errors"
)

var (
	// ErrServerNotFound error returned when no server is found
	ErrServerNotFound = errors.New("server not found")
)

// Config generic auth configuration
type Config struct {
	Servers       []Server `json:"servers"`
	CurrentServer string   `json:"currentserver"`
}

// urlsEqual checks if two URL are equal
func urlsEqual(url1, url2 string) bool {
	return url1 == url2 || strings.TrimSuffix(url1, "/") == strings.TrimSuffix(url2, "/")
}

// GetUsers returns all users auth of a server
func (c *Config) GetUsers(serverURL string) ([]User, error) {
	result := []User{}
	for _, server := range c.Servers {
		if urlsEqual(server.URL, serverURL) {
			result = append(result, server.Users...)
			return result, nil
		}
	}
	return result, ErrServerNotFound
}

// GetUser retuns the username of a server if found, otherwise an error
func (c *Config) GetUser(serverURL string, username string) (User, error) {
	if username == "" {
		return User{}, errors.New("empty username")
	}
	users, err := c.GetUsers(serverURL)
	if err != nil {
		return User{}, err
	}
	for _, user := range users {
		if user.Username == username {
			return user, nil
		}
	}
	return User{}, ErrUserNotFound
}

// IndexOfServerName returns the index of a server
func (c *Config) IndexOfServerName(name string) int {
	for i, server := range c.Servers {
		if server.Name == name {
			return i
		}
	}
	return -1
}

// GetServerByName returns the server for the given URL
func (c *Config) GetServer(url string) (Server, error) {
	for _, s := range c.Servers {
		if urlsEqual(s.URL, url) {
			return s, nil
		}
	}
	return Server{}, ErrServerNotFound
}

// GetServerByName returns the server for the given name
func (c *Config) GetServerByName(name string) (Server, error) {
	for _, s := range c.Servers {
		if s.Name == name {
			return s, nil
		}
	}
	return Server{}, ErrServerNotFound
}

// GetServerByKind returns the server for the given kind
func (c *Config) GetServerByKind(kind string) (Server, error) {
	for _, s := range c.Servers {
		if s.Kind == kind {
			return s, nil
		}
	}
	return Server{}, ErrServerNotFound
}

//DeleteServer deletes the server for the given URL and updates the current server
//if is the same with the deleted server
func (c *Config) DeleteServer(url string) {
	for i, s := range c.Servers {
		if urlsEqual(s.URL, url) {
			c.Servers = append(c.Servers[:i], c.Servers[i+1:]...)
		}
	}
	if urlsEqual(c.CurrentServer, url) && len(c.Servers) > 0 {
		c.CurrentServer = c.Servers[0].URL
	} else {
		c.CurrentServer = ""
	}
}

// GetCurrentServer returns the current srver authentication
func (c *Config) GetCurrentServer() (Server, error) {
	return c.GetServer(c.CurrentServer)
}

// SetCurrentServer configures the current server
func (c *Config) SetCurrentServer(url string) error {
	server, err := c.GetServer(url)
	if err != nil {
		return err
	}
	c.CurrentServer = server.URL
	return nil
}

// GetServesrNames returns the names of the servers currently in the configuration
func (c *Config) GetServersNames() []string {
	answer := []string{}
	for _, server := range c.Servers {
		name := server.Name
		if name != "" {
			answer = append(answer, name)
		}
	}
	sort.Strings(answer)
	return answer
}

// GetServersURLs returns the servesr URLs currently in the configuration
func (c *Config) GetServerURLs() []string {
	answer := []string{}
	for _, server := range c.Servers {
		u := server.URL
		if u != "" {
			answer = append(answer, u)
		}
	}
	sort.Strings(answer)
	return answer
}

// AddUserToServer adds a new user to an existing server
func (c *Config) AddUserToServer(url string, user User) error {
	for i, s := range c.Servers {
		if urlsEqual(s.URL, url) {
			users := c.Servers[i].Users
			// overwrites an existing user
			for j, u := range users {
				if u.Username == user.Username {
					users[j] = user
					return nil
				}
			}
			c.Servers[i].Users = append(c.Servers[i].Users, user)
			return nil
		}
	}
	return ErrServerNotFound
}

// AddServer adds  a new server without any user authentications. If the server exists and the overwrite flag is set, the
// server will be overwritten.
func (c *Config) AddServer(url string, name string, kind string, overwrite bool) {
	server := Server{
		URL:  url,
		Name: name,
		Kind: kind,
	}
	for i, s := range c.Servers {
		if urlsEqual(s.URL, url) {
			if overwrite {
				c.Servers[i] = server
			}
			return
		}
	}

	c.Servers = append(c.Servers, server)
}
