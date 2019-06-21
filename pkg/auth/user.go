package auth

import (
	"os"
	"strings"
)

const (
	usernameSuffix    = "_USERNAME"
	apiTokenSuffix    = "_API_TOKEN"
	bearerTokenSuffix = "_BEARER_TOKEN"
	DefaultUsername   = "dummy"
)

// User generic auth credentials for a user
type User struct {
	Username    string `json:"username"`
	ApiToken    string `json:"apitoken"`
	BearerToken string `json:"bearertoken"`
	Password    string `json:"password,omitempty"`
}

// usernameEnv builds the username environment variable name
func usernameEnv(prefix string) string {
	prefix = strings.ToUpper(prefix)
	return prefix + usernameSuffix
}

// apiTokenEnv builds the api token environment variable name
func apiTokenEnv(prefix string) string {
	prefix = strings.ToUpper(prefix)
	return prefix + apiTokenSuffix
}

// bearerTokenEnv builds the bearer token environment variable name
func bearerTokenEnv(prefix string) string {
	prefix = strings.ToUpper(prefix)
	return prefix + bearerTokenSuffix
}

// CreateAuthUserFromEnvironment creates a user auth from environment variables
func CreateAuthUserFromEnvironment(prefix string) User {
	user := User{}
	username, set := os.LookupEnv(usernameEnv(prefix))
	if set {
		user.Username = username
	}
	apiToken, set := os.LookupEnv(apiTokenEnv(prefix))
	if set {
		user.ApiToken = apiToken
	}
	bearerToken, set := os.LookupEnv(bearerTokenEnv(prefix))
	if set {
		user.BearerToken = bearerToken
	}

	if user.ApiToken != "" || user.Password != "" {
		if user.Username == "" {
			user.Username = DefaultUsername
		}
	}

	return user
}

// IsInvalid returns true if the user auth has a valid token
func (u User) IsInvalid() bool {
	return u.BearerToken == "" && (u.ApiToken == "" || u.Username == "")
}
