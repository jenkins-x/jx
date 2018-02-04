package auth

import (
	"os"
)

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
