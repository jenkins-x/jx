package auth

import (
	"errors"
	"fmt"
)

//UserKind define for user kind
type UserKind string

const (
	// UserKindLocal indicates a user of type local
	UserKindLocal UserKind = "local"
	// UserKindPipeline indicates a user of type pipeline (e.g. used by the build within the pipeline
	UserKindPipeline UserKind = "pipeline"
)

//User store the auth infomation for a user
type User struct {
	Username    string   `json:"username"`
	ApiToken    string   `json:"apitoken"`
	BearerToken string   `json:"bearertoken"`
	Password    string   `json:"password,omitempty"`
	Kind        UserKind `json:"kind"`
}

//ServerKind type of the server
type ServerKind string

const (
	// ServerKindGit idicates a server configuration for git
	ServerKindGit ServerKind = "git"
	// ServerKindIssue indicates a server configuration for issue
	ServerKindIssue ServerKind = "issue"
	// ServerKindChat idicates a server configuration for chat
	ServerKindChat ServerKind = "chat"
)

//ServiceKind type for service used by the server
type ServiceKind string

const (
	// ServiceKindGithub indicates that the git server is using as service GitHub
	ServiceKindGithub ServiceKind = "github"
	// ServiceKindGitlab indicates that the git server is using as service Gitlab
	ServiceKindGitlab ServiceKind = "gitlab"
	// ServiceKindGitea indicates that the git server is using as service Gitea
	ServiceKindGitea ServiceKind = "gitea"
	// ServiceKindBitbucketCloud indicates that the git server is using as service Bitbucket Cloud
	ServiceKindBitbucketCloud ServiceKind = "bitbucketcloud"
	// ServiceKindBitbucketServer indicates that the git server is using as service Bitbuckst Server
	ServiceKindBitbucketServer ServiceKind = "bitbucketserver"
)

//Server stores the server configuration for a server
type Server struct {
	URL         string      `json:"url"`
	Name        string      `json:"name"`
	Kind        ServerKind  `json:"kind"`
	ServiceKind ServiceKind `json:"servicekind"`
	Users       []*User     `json:"users"`
}

// Config stores the entire auth configuration for a number of sservers
type Config struct {
	Servers []*Server `json:"servers"`
}

// Checker verifies if the configuration is valid
type Checker interface {
	Valid() error
}

// Valid checks if the user config is valid
func (u *User) Valid() error {
	if u.BearerToken != "" {
		return nil
	}
	if u.Username == "" {
		return errors.New("Empty username")
	}
	if u.ApiToken == "" && u.Password == "" {
		return errors.New("Empty API token and password")
	}
	return nil

}

// Valid checks if a server config is valid
func (s *Server) Valid() error {
	if len(s.Users) == 0 {
		return fmt.Errorf("%s server has no users", s.Name)
	}
	if s.URL == "" {
		return fmt.Errorf("%s server has an empty URL", s.Name)
	}
	for _, u := range s.Users {
		err := u.Valid()
		if err != nil {
			return err
		}
	}
	return nil
}

// PipelineUser returns the pipeline user, if there is not pipeline user available
// returns the first user
func (s *Server) PipelineUser() User {
	for _, user := range s.Users {
		if user.Kind == UserKindPipeline {
			return *user
		}
	}
	if len(s.Users) > 0 {
		return *s.Users[0]
	}
	return User{}
}

// Valid checks if the config is valid
func (c *Config) Valid() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("No servers in config")
	}
	for _, s := range c.Servers {
		err := s.Valid()
		if err != nil {
			return err
		}
	}
	return nil
}
