package auth

import (
	"sort"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// ErrUserNotFound error returned when no user is found
var ErrUserNotFound = errors.New("user not found")

// Server configuration for a generic server authentication
type Server struct {
	URL   string `json:"url"`
	Users []User `json:"users"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`

	CurrentUser string `json:"currentuser"`
}

// Label returns the label of the server
func (s Server) Label() string {
	if s.Name != "" {
		return s.Name
	}
	return s.URL
}

// Description returns the server description
func (s Server) Description() string {
	if s.Name != "" {
		return s.Name + " at " + s.URL
	}
	return s.URL
}

// DeleteUser delete a user from the server configuration
func (s Server) DeleteUser(username string) error {
	idx := -1
	for i, user := range s.Users {
		if user.Username == username {
			idx = i
			break
		}
	}
	if idx < 0 {
		if len(s.Users) == 0 {
			return ErrUserNotFound
		}
		return util.InvalidArg(username, s.GetUsernames())
	}
	s.Users = append(s.Users[0:idx], s.Users[idx+1:]...)
	return nil
}

// GetUsernames returns all user names currently configured
func (s Server) GetUsernames() []string {
	answer := []string{}
	for _, user := range s.Users {
		name := user.Username
		if name != "" {
			answer = append(answer, name)
		}
	}
	sort.Strings(answer)
	return answer
}

// HasUsers checks if a server has any user auth configured
func (s Server) HasUsers() bool {
	return len(s.Users) > 0
}

// GetUser returns the user auth by user name
func (s Server) GetUser(username string) (User, error) {
	for _, user := range s.Users {
		if username == user.Username {
			return user, nil
		}
	}
	return User{}, ErrUserNotFound
}

// GetCurrentUser returns the current user
func (s Server) GetCurrentUser() (User, error) {
	return s.GetUser(s.CurrentUser)
}

// SetCurrentUser configure the current user by user name
func (s Server) SetCurrentUser(username string) error {
	if s.CurrentUser == username {
		return nil
	}
	for _, u := range s.Users {
		if u.Username == username {
			s.CurrentUser = username
			return nil
		}
	}
	return ErrUserNotFound
}

// AddUser adds a new user to the server configuration. It overwrites the old user it case it exists.
func (s Server) AddUser(user User) {
	// overwrite an existing user
	for i, u := range s.Users {
		if u.Username == user.Username {
			s.Users[i] = user
			return
		}
	}
	s.Users = append(s.Users, user)
}
