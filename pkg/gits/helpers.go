package gits

import (
	"os"
	"os/user"

	"github.com/pkg/errors"
)

// EnsureUserAndEmailSetup returns the user name and email for the gitter
// lazily setting them if they are blank either from the environment variables
// `GIT_AUTHOR_NAME` and `GIT_AUTHOR_EMAIL` or using default values
func EnsureUserAndEmailSetup(gitter Gitter) (string, string, error) {
	userName, _ := gitter.Username("")
	userEmail, _ := gitter.Email("")
	if userName == "" {
		userName = os.Getenv("GIT_AUTHOR_NAME")
		if userName == "" {
			user, err := user.Current()
			if err == nil && user != nil {
				userName = user.Username
			}
		}
		if userName == "" {
			userName = "jenkins-x-bot"
		}
		err := gitter.SetUsername("", userName)
		if err != nil {
			return userName, userEmail, errors.Wrapf(err, "Failed to set the git username to %s", userName)
		}
	}
	if userEmail == "" {
		userEmail = os.Getenv("GIT_AUTHOR_EMAIL")
		if userEmail == "" {
			userEmail = "jenkins-x@googlegroups.com"
		}
		err := gitter.SetEmail("", userEmail)
		if err != nil {
			return userName, userEmail, errors.Wrapf(err, "Failed to set the git email to %s", userEmail)
		}
	}
	return userName, userEmail, nil
}
