package gits

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strings"

	uuid "github.com/satori/go.uuid"

	"github.com/jenkins-x/jx/pkg/auth"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/log"

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
		log.Infof("Converted %s to an unshallow repository\n", dir)
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
				log.Infof("ran %s in %s\n", util.ColorInfo("git remote update"), dir)
			}
		}
		if verbose {
			log.Infof("ran git fetch %s %s in %s\n", remote, strings.Join(refspecs, " "), dir)
		}
		err = Unshallow(dir, gitter)
		if err != nil {
			return errors.WithStack(err)
		}
		if verbose {
			log.Infof("Unshallowed git repo in %s\n", dir)
		}
	} else {
		if verbose {
			log.Infof("ran git fetch --unshallow %s %s in %s\n", remote, strings.Join(refspecs, " "), dir)
		}
	}
	branches, err := gitter.LocalBranches(dir)
	if err != nil {
		return errors.Wrapf(err, "listing local branches")
	}
	found := false
	for _, b := range branches {
		if b == baseBranch {
			found = true
			break
		}
	}
	if !found {
		err = gitter.CreateBranch(dir, baseBranch)
		if err != nil {
			return errors.Wrapf(err, "creating branch %s", baseBranch)
		}
	}
	// Ensure we are on baseBranch
	err = gitter.Checkout(dir, baseBranch)
	if err != nil {
		return errors.Wrapf(err, "checking out %s", baseBranch)
	}
	if verbose {
		log.Infof("ran git checkout %s in %s\n", baseBranch, dir)
	}
	// Ensure we are on the right revision
	err = gitter.ResetHard(dir, baseSha)
	if err != nil {
		return errors.Wrapf(err, "resetting %s to %s", baseBranch, baseSha)
	}
	if verbose {
		log.Infof("ran git reset --hard %s in %s\n", baseSha, dir)
	}
	err = gitter.CleanForce(dir, ".")
	if err != nil {
		return errors.Wrapf(err, "cleaning up the git repo")
	}
	if verbose {
		log.Infof("ran clean --force -d . in %s\n", dir)
	}
	// Now do the merges
	for _, sha := range SHAs {
		err := gitter.Merge(dir, sha)
		if err != nil {
			return errors.Wrapf(err, "merging %s into master", sha)
		}
		if verbose {
			log.Infof("ran git merge %s in %s\n", sha, dir)
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
		log.Warnf("failed to parse git provider URL %s: %s\n", text, err.Error())
		return text
	}
	u.Path = ""
	if !strings.HasPrefix(u.Scheme, "http") {
		// lets convert other schemes like 'git' to 'https'
		u.Scheme = "https"
	}
	return u.String()
}

// PushRepoAndCreatePullRequest commits and pushes the changes in the repo rooted at dir.
// It creates a branch called branchName from a base.
// It uses the pullRequestDetails for the message and title for the commit and PR.
// It uses and updates pullRequestInfo to identify whether to rebase an existing PR.
func PushRepoAndCreatePullRequest(dir string, gitInfo *GitRepository, base string, prDetails *PullRequestDetails,
	prInfo *PullRequestInfo, fork bool, commit bool, push bool, provider GitProvider, gitter Gitter) (*PullRequestInfo, error) {
	if commit {

		err := gitter.Add(dir, "-A")
		if err != nil {
			return nil, errors.WithStack(err)
		}
		changed, err := gitter.HasChanges(dir)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if !changed {
			log.Warnf("No changes made to the source code in %s. Code must be up to date!", dir)
			return nil, nil
		}
		err = gitter.CommitDir(dir, prDetails.Message)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	if push {
		// lets rebase an existing PR
		if prInfo != nil && prInfo.PullRequestArguments.Head != "" {
			err := gitter.ForcePushBranch(dir, prDetails.BranchName, prInfo.PullRequestArguments.Head)
			if err != nil {
				return nil, errors.Wrapf(err, "rebasing existing PR on %s", prInfo.PullRequestArguments.Head)
			}
		}

		err := gitter.Push(dir)
		if err != nil {
			return nil, err
		}
	}

	headPrefix := ""

	username := provider.CurrentUsername()
	if username == "" {
		return nil, fmt.Errorf("no git user name found")
	}
	if gitInfo.Organisation != username && fork {
		headPrefix = username + ":"
	}

	gha := &GitPullRequestArguments{
		GitRepository: gitInfo,
		Title:         prDetails.Title,
		Body:          prDetails.Message,
		Base:          base,
		Head:          headPrefix + prDetails.BranchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Infof("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return &PullRequestInfo{
		GitProvider:          provider,
		PullRequest:          pr,
		PullRequestArguments: gha,
	}, nil
}

// ForkAndPullPullRepo pulls the specified gitUrl into baseDir/org/repo using gitter, creating a remote fork if needed using the git provider
func ForkAndPullPullRepo(gitURL string, baseDir string, baseRef string, branchName string, provider GitProvider, gitter Gitter, configGitFn ConfigureGitFn) (string, string, *GitRepository, bool, error) {
	fork := false
	if gitURL == "" {
		return "", "", nil, fork, fmt.Errorf("No source gitter URL")
	}
	gitInfo, err := ParseGitURL(gitURL)
	if err != nil {
		return "", "", nil, fork, errors.Wrapf(err, "failed to parse gitter URL %s", gitURL)
	}

	username := ""
	userDetails := auth.UserAuth{}
	originalOrg := gitInfo.Organisation
	originalRepo := gitInfo.Name

	if provider == nil {
		log.Warnf("No GitProvider specified!\n")
		debug.PrintStack()
	} else {
		userDetails = provider.UserAuth()
		username = provider.CurrentUsername()

		// lets check if we need to fork the repository...
		if originalOrg != username && username != "" && originalOrg != "" && provider.ShouldForkForPullRequest(originalOrg, originalRepo, username) {
			fork = true
		}
	}

	dir := filepath.Join(baseDir, gitInfo.Organisation, gitInfo.Name)

	if baseRef == "" {
		baseRef = "master"
	}

	if fork {
		if provider == nil {
			return "", "", nil, fork, errors.Wrapf(err, "no Git Provider specified for gitter URL %s", gitURL)
		}
		repo, err := provider.GetRepository(username, originalRepo)
		if err != nil {
			// lets try create a fork - using a blank organisation to force a user specific fork
			repo, err = provider.ForkRepository(originalOrg, originalRepo, "")
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "failed to fork GitHub repo %s/%s to user %s", originalOrg, originalRepo, username)
			}
			log.Infof("Forked Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
		}

		// lets only use this repository if it is a fork
		if !repo.Fork {
			fork = false
		} else {
			dir, err = ioutil.TempDir("", fmt.Sprintf("fork-%s-%s", gitInfo.Organisation, gitInfo.Name))
			if err != nil {
				return "", "", nil, fork, errors.Wrap(err, "failed to create temp dir")
			}

			err = os.MkdirAll(dir, util.DefaultWritePermissions)
			if err != nil {
				return "", "", nil, fork, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
			}
			cloneGitURL, err := gitter.CreatePushURL(repo.CloneURL, &userDetails)
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "failed to get clone URL from %s and user %s", repo.CloneURL, username)
			}
			err = gitter.Clone(cloneGitURL, dir)
			if err != nil {
				return "", "", nil, fork, errors.WithStack(err)
			}
			err = gitter.FetchBranch(dir, "origin")
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "fetching from %s", cloneGitURL)
			}
			err = gitter.SetRemoteURL(dir, "upstream", gitURL)
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "setting remote upstream %q in forked environment repo", gitURL)
			}
			if configGitFn != nil {
				err = configGitFn(dir, gitInfo, gitter)
				if err != nil {
					return "", "", nil, fork, errors.WithStack(err)
				}
			}
			branchName, err := computeBranchName(baseRef, branchName, dir, gitter)
			if err != nil {
				return "", "", nil, fork, errors.WithStack(err)
			}
			if branchName != "master" {
				err = gitter.CreateBranch(dir, branchName)
				if err != nil {
					return "", "", nil, fork, errors.WithStack(err)
				}

				err = gitter.Checkout(dir, branchName)
				if err != nil {
					return "", "", nil, fork, errors.WithStack(err)
				}
			}
			err = gitter.ResetToUpstream(dir, baseRef)
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "resetting forked branch %s to upstream version", baseRef)
			}
			return dir, baseRef, gitInfo, fork, nil
		}
	}

	// now lets clone the fork and pull it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return "", "", nil, fork, errors.Wrapf(err, "failed to check if directory %s exists", dir)
	}

	if exists {
		if configGitFn != nil {
			err = configGitFn(dir, gitInfo, gitter)
			if err != nil {
				return "", "", nil, fork, errors.WithStack(err)
			}
		}
		// lets check the gitter remote URL is setup correctly
		err = gitter.SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return "", "", nil, fork, errors.WithStack(err)
		}
		err = gitter.Stash(dir)
		if err != nil {
			return "", "", nil, fork, errors.WithStack(err)
		}
		branchName, err := computeBranchName(baseRef, branchName, dir, gitter)
		if err != nil {
			return "", "", nil, fork, errors.WithStack(err)
		}
		if branchName != "master" {
			err = gitter.CreateBranch(dir, branchName)
			if err != nil {
				return "", "", nil, fork, errors.WithStack(err)
			}
		}
		err = gitter.Checkout(dir, branchName)
		if err != nil {
			return "", "", nil, fork, errors.WithStack(err)
		}
		err = gitter.FetchBranch(dir, "orgin", baseRef)
		if err != nil {
			return "", "", nil, fork, errors.WithStack(err)
		}
		err = gitter.Merge(dir, baseRef)
		if err != nil {
			return "", "", nil, fork, errors.WithStack(err)
		}
	} else {
		err := os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return "", "", nil, fork, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		cloneGitURL, err := gitter.CreatePushURL(gitURL, &userDetails)
		if err != nil {
			return "", "", nil, fork, errors.Wrapf(err, "failed to get clone URL from %s and user %s", gitURL, username)
		}

		err = gitter.Clone(cloneGitURL, dir)
		if err != nil {
			return "", "", nil, fork, errors.WithStack(err)
		}
		if configGitFn != nil {
			err = configGitFn(dir, gitInfo, gitter)
			if err != nil {
				return "", "", nil, fork, errors.WithStack(err)
			}
		}
		branchName, err := computeBranchName(baseRef, branchName, dir, gitter)
		if err != nil {
			return "", "", nil, fork, errors.WithStack(err)
		}
		if branchName != "master" {
			err = gitter.CreateBranch(dir, branchName)
			if err != nil {
				return "", "", nil, fork, errors.WithStack(err)
			}

			err = gitter.Checkout(dir, branchName)
			if err != nil {
				return "", "", nil, fork, errors.WithStack(err)
			}
		}
	}

	return dir, baseRef, gitInfo, fork, nil
}

func computeBranchName(baseRef string, branchName string, dir string, gitter Gitter) (string, error) {

	if branchName == "" {
		branchName = baseRef
	}
	validBranchName := gitter.ConvertToValidBranchName(branchName)

	branchNames, err := gitter.RemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return "", errors.Wrapf(err, "Failed to load remote branch names")
	}
	if util.StringArrayIndex(branchNames, validBranchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchNameUUID, err := uuid.NewV4()
		if err != nil {
			return "", errors.WithStack(err)
		}
		validBranchName += "-" + branchNameUUID.String()
	}
	return validBranchName, nil
}
