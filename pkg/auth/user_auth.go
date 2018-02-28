package auth

import (
	"os"
)

// CreateAuthUserFromEnvironment creates a user auth from environment vairables
func CreateAuthUserFromEnvironment(prefix string) UserAuth {
	return UserAuth{
		Username:    os.Getenv(prefix + "_USERNAME"),
		ApiToken:    os.Getenv(prefix + "_API_TOKEN"),
		BearerToken: os.Getenv(prefix + "_BEARER_TOKEN"),
	}
}

// IsInvalid returns true if the user auth has a valid token
func (a *UserAuth) IsInvalid() bool {
	return a.BearerToken == "" && (a.ApiToken == "" || a.Username == "")
}
