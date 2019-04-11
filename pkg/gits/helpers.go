package gits

import (
	"fmt"
	"net/url"
	"os"
	"os/user"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/sirupsen/logrus"

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

// Unshallow converts a shallow git repo (one cloned with --depth=n) into one has full depth for the current branch
// and all tags. Note that remote branches are still not fetched,
// you need to do this manually. It checks if the repo is shallow or not before trying to unshallow it.
func Unshallow(dir string, gitter Gitter) error {
	shallow, err := gitter.IsShallow(dir)
	if err != nil {
		return err
	}
	if shallow {
		if err := gitter.FetchUnshallow(dir); err != nil {
			return err
		}
		if err := gitter.FetchTags(dir); err != nil {
			return err
		}
		logrus.Infof("Converted %s to an unshallow repository\n", dir)
		return nil
	}
	return nil
}

// FetchAndMergeSHAs merges any SHAs into the baseBranch which has a tip of baseSha,
// fetching the commits from remote for the git repo in dir. It will try to fetch individual commits (
// if the remote repo supports it - see https://github.
// com/git/git/commit/68ee628932c2196742b77d2961c5e16360734a62) otherwise it uses git remote update to pull down the
// whole repo.
func FetchAndMergeSHAs(SHAs []string, baseBranch string, baseSha string, remote string, dir string,
	gitter Gitter, verbose bool) error {
	refspecs := make([]string, 0)
	for _, sha := range SHAs {
		refspecs = append(refspecs, fmt.Sprintf("%s:", sha))
	}
	refspecs = append(refspecs, fmt.Sprintf("%s:", baseSha))
	// First lets make sure we have the commits - remember that this may be a shallow clone
	err := gitter.FetchBranchUnshallow(dir, remote, refspecs...)
	if err != nil {
		// Unshallow fetch failed, so do a full unshallow
		// First ensure we actually have the branch refs
		err := gitter.FetchBranch(dir, remote, refspecs...)
		if err != nil {
			// This can be caused by git not being configured to allow fetching individual SHAs
			// There is not a nice way to solve this except to attempt to do a full fetch
			err = gitter.RemoteUpdate(dir)
			if err != nil {
				return errors.Wrapf(err, "updating remote %s", remote)
			}
			if verbose {
				logrus.Infof("ran %s in %s\n", util.ColorInfo("git remote update"), dir)
			}
		}
		if verbose {
			logrus.Infof("ran git fetch %s %s in %s\n", remote, strings.Join(refspecs, " "), dir)
		}
		err = Unshallow(dir, gitter)
		if err != nil {
			return errors.WithStack(err)
		}
		if verbose {
			logrus.Infof("Unshallowed git repo in %s\n", dir)
		}
	} else {
		if verbose {
			logrus.Infof("ran git fetch --unshallow %s %s in %s\n", remote, strings.Join(refspecs, " "), dir)
		}
	}

	// Ensure we are on baseBranch
	err = gitter.Checkout(dir, baseBranch)
	if err != nil {
		return errors.Wrapf(err, "checking out %s", baseBranch)
	}
	if verbose {
		logrus.Infof("ran git checkout %s in %s\n", baseBranch, dir)
	}
	// Ensure we are on the right revision
	err = gitter.ResetHard(dir, baseSha)
	if err != nil {
		errors.Wrapf(err, "resetting %s to %s", baseBranch, baseSha)
	}
	if verbose {
		logrus.Infof("ran git reset --hard %s in %s\n", baseSha, dir)
	}
	err = gitter.CleanForce(dir, ".")
	if err != nil {
		return errors.Wrapf(err, "cleaning up the git repo")
	}
	if verbose {
		logrus.Infof("ran clean --force -d . in %s\n", dir)
	}
	// Now do the merges
	for _, sha := range SHAs {
		err := gitter.Merge(dir, sha)
		if err != nil {
			return errors.Wrapf(err, "merging %s into master", sha)
		}
		if verbose {
			logrus.Infof("ran git merge %s in %s\n", sha, dir)
		}
	}
	return nil
}

// SourceRepositoryProviderURL returns the git provider URL for the SourceRepository which is something like
// either `https://hostname` or `http://hostname`
func SourceRepositoryProviderURL(gitProvider GitProvider) string {
	return GitProviderURL(gitProvider.ServerURL())
}

// GitProviderURL returns the git provider host URL for the SourceRepository which is something like
// either `https://hostname` or `http://hostname`
func GitProviderURL(text string) string {
	if text == "" {
		return text
	}
	u, err := url.Parse(text)
	if err != nil {
		logrus.Warnf("failed to parse git provider URL %s: %s\n", text, err.Error())
		return text
	}
	u.Path = ""
	if !strings.HasPrefix(u.Scheme, "http") {
		// lets convert other schemes like 'git' to 'https'
		u.Scheme = "https"
	}
	return u.String()
}
