package gits

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"sort"
	"strings"

	uuid "github.com/satori/go.uuid"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/pkg/errors"
)

const (
	// LabelUpdatebot is the label applied to PRs created by updatebot
	LabelUpdatebot = "updatebot"
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
		log.Logger().Infof("Converted %s to an unshallow repository", dir)
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
	gitter Gitter) error {
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
			log.Logger().Debugf("ran %s in %s", util.ColorInfo("git remote update"), dir)
		}
		log.Logger().Debugf("ran git fetch %s %s in %s", remote, strings.Join(refspecs, " "), dir)

		err = Unshallow(dir, gitter)
		if err != nil {
			return errors.WithStack(err)
		}
		log.Logger().Debugf("Unshallowed git repo in %s", dir)
	} else {
		log.Logger().Debugf("ran git fetch --unshallow %s %s in %s", remote, strings.Join(refspecs, " "), dir)
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
	log.Logger().Debugf("ran git checkout %s in %s", baseBranch, dir)
	// Ensure we are on the right revision
	err = gitter.Reset(dir, baseSha, true)
	if err != nil {
		return errors.Wrapf(err, "resetting %s to %s", baseBranch, baseSha)
	}
	log.Logger().Debugf("ran git reset --hard %s in %s", baseSha, dir)
	err = gitter.CleanForce(dir, ".")
	if err != nil {
		return errors.Wrapf(err, "cleaning up the git repo")
	}
	log.Logger().Debugf("ran clean --force -d . in %s", dir)
	// Now do the merges
	for _, sha := range SHAs {
		err := gitter.Merge(dir, sha)
		if err != nil {
			return errors.Wrapf(err, "merging %s into master", sha)
		}
		log.Logger().Debugf("ran git merge %s in %s", sha, dir)

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
		log.Logger().Warnf("failed to parse git provider URL %s: %s", text, err.Error())
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
func PushRepoAndCreatePullRequest(dir string, upstreamRepo *GitRepository, forkRepo *GitRepository, base string, prDetails *PullRequestDetails, filter *PullRequestFilter, commit bool, commitMessage string, push bool, dryRun bool, gitter Gitter, provider GitProvider, labels []string) (*PullRequestInfo, error) {
	userAuth := provider.UserAuth()
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
			log.Logger().Warnf("No changes made to the source code in %s. Code must be up to date!", dir)
			return nil, nil
		}
		if commitMessage == "" {
			commitMessage = prDetails.Message
		}
		err = gitter.CommitDir(dir, commitMessage)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	headPrefix := ""

	username := upstreamRepo.Organisation
	cloneURL := upstreamRepo.CloneURL
	if forkRepo != nil {
		username = forkRepo.Organisation
		cloneURL = forkRepo.CloneURL
	}

	if upstreamRepo.Organisation != username {
		headPrefix = username + ":"
	}

	gha := &GitPullRequestArguments{
		GitRepository: upstreamRepo,
		Title:         prDetails.Title,
		Body:          prDetails.Message,
		Base:          base,
	}
	var existingPr *GitPullRequest

	forkPushURL, err := gitter.CreateAuthenticatedURL(cloneURL, &userAuth)
	if err != nil {
		return nil, errors.Wrapf(err, "creating push URL for %s", cloneURL)
	}

	if filter != nil && push {
		// lets rebase an existing PR
		existingPrs, err := FilterOpenPullRequests(provider, upstreamRepo.Organisation, upstreamRepo.Name, *filter)
		if err != nil {
			return nil, errors.Wrapf(err, "finding existing PRs using filter %s on repo %s/%s", filter.String(), upstreamRepo.Organisation, upstreamRepo.Name)
		}

		if len(existingPrs) > 1 {
			sort.SliceStable(existingPrs, func(i, j int) bool {
				// sort in descending order of PR numbers (assumes PRs numbers increment!)
				return util.DereferenceInt(existingPrs[j].Number) < util.DereferenceInt(existingPrs[i].Number)
			})
			prs := make([]string, 0)
			for _, pr := range existingPrs {
				prs = append(prs, pr.URL)
			}
			log.Logger().Debugf("Found more than one PR %s using filter %s on repo %s/%s so rebasing latest PR %s", strings.Join(prs, ", "), filter.String(), upstreamRepo.Organisation, upstreamRepo.Name, existingPrs[:1][0].URL)
			existingPr = existingPrs[0]
		} else if len(existingPrs) == 1 {
			existingPr = existingPrs[0]
		}
	}
	remoteBranch := prDetails.BranchName
	if existingPr != nil {
		if util.DereferenceString(existingPr.HeadOwner) == username && existingPr.HeadRef != nil && existingPr.Number != nil {
			remote := "origin"
			if forkRepo != nil && forkRepo.Fork {
				remote = "upstream"
			}
			changeBranch, err := gitter.Branch(dir)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			localBranchUUID, err := uuid.NewV4()
			if err != nil {
				return nil, errors.Wrapf(err, "creating UUID for local branch")
			}
			// We use this "dummy" local branch to pull into to avoid having to work with FETCH_HEAD as our local
			// representation of the remote branch. This is an oddity of the pull/%d/head remote.
			localBranch := localBranchUUID.String()
			remoteBranch = *existingPr.HeadRef
			fetchRefSpec := fmt.Sprintf("pull/%d/head:%s", *existingPr.Number, localBranch)
			err = gitter.FetchBranch(dir, remote, fetchRefSpec)
			if err != nil {
				return nil, errors.Wrapf(err, "fetching %s for merge", fetchRefSpec)
			}

			err = gitter.CreateBranchFrom(dir, prDetails.BranchName, localBranch)
			if err != nil {
				return nil, errors.Wrapf(err, "creating branch %s from %s", prDetails.BranchName, fetchRefSpec)
			}
			err = gitter.Checkout(dir, prDetails.BranchName)
			if err != nil {
				return nil, errors.Wrapf(err, "checking out branch %s", prDetails.BranchName)
			}
			err = gitter.MergeTheirs(dir, changeBranch)
			if err != nil {
				return nil, errors.Wrapf(err, "merging %s into %s", changeBranch, fetchRefSpec)
			}
			err = gitter.RebaseTheirs(dir, fmt.Sprintf(localBranch), "", true)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		} else {
			// We can only update an existing PR if the owner of that PR is this user, so we clear the existingPr
			existingPr = nil
		}
	}
	if dryRun {
		log.Logger().Infof("Commit created but not pushed; would have updated pull request %s with %s and used commit message %s. Please manually delete %s when you are done", util.ColorInfo(existingPr.URL), prDetails.String(), commitMessage, util.ColorInfo(dir))
		return nil, nil
	} else if push {
		err := gitter.Push(dir, forkPushURL, true, false, fmt.Sprintf("%s:%s", "HEAD", remoteBranch))
		if err != nil {
			return nil, errors.Wrapf(err, "pushing merged branch %s", remoteBranch)
		}
	}
	var pr *GitPullRequest
	if existingPr != nil {
		gha.Head = headPrefix + remoteBranch
		// work out the minimal similar title
		if strings.HasPrefix(existingPr.Title, "chore(deps): bump ") {
			origWords := strings.Split(existingPr.Title, " ")
			newWords := strings.Split(prDetails.Title, " ")
			answer := make([]string, 0)
			for i, w := range newWords {
				if len(origWords) > i && origWords[i] == w {
					answer = append(answer, w)
				}
			}
			if answer[len(answer)-1] == "bump" {
				// if there are no similarities in the actual dependency, then add a generic form of words
				answer = append(answer, "dependency", "versions")
			}
			if answer[len(answer)-1] == "to" || answer[len(answer)-1] == "from" {
				// remove trailing prepositions
				answer = answer[:len(answer)-1]
			}
			gha.Title = strings.Join(answer, " ")
		} else {
			gha.Title = prDetails.Title
		}
		gha.Body = fmt.Sprintf("%s\n<hr />\n\n%s", prDetails.Message, existingPr.Body)
		var err error
		pr, err = provider.UpdatePullRequest(gha, *existingPr.Number)
		if err != nil {
			return nil, errors.Wrapf(err, "updating pull request %s", existingPr.URL)
		}
		log.Logger().Infof("Updated Pull Request: %s", util.ColorInfo(pr.URL))
	} else {
		gha.Head = headPrefix + prDetails.BranchName

		pr, err = provider.CreatePullRequest(gha)
		if err != nil {
			return nil, errors.Wrapf(err, "creating pull request with arguments %v", gha.String())
		}
		log.Logger().Infof("Created Pull Request: %s", util.ColorInfo(pr.URL))
	}
	if len(labels) > 0 {
		number := *pr.Number
		var err error
		err = provider.AddLabelsToIssue(pr.Owner, pr.Repo, number, labels)
		if err != nil {
			return nil, err
		}
		log.Logger().Infof("Added label %s to Pull Request %s", util.ColorInfo(strings.Join(labels, ", ")), pr.URL)
	}
	return &PullRequestInfo{
		GitProvider:          provider,
		PullRequest:          pr,
		PullRequestArguments: gha,
	}, nil
}

// ForkAndPullRepo pulls the specified gitUrl into dir using gitter, creating a remote fork if needed using the git provider
//
// If there are existing files in dir (and dir is already a git clone), the existing files will pushed into the stash
// and then popped at the end. If they cannot be popped then an error will be returned which can be checked for using
// IsCouldNotPopTheStashError
func ForkAndPullRepo(gitURL string, dir string, baseRef string, branchName string, provider GitProvider, gitter Gitter, forkName string) (string, string, *GitRepository, *GitRepository, error) {
	// Validate the arguments
	if gitURL == "" {
		return "", "", nil, nil, fmt.Errorf("gitURL cannot be nil")
	}
	if provider == nil {
		return "", "", nil, nil, errors.Errorf("provider cannot be nil for %s", gitURL)
	}
	originalInfo, err := ParseGitURL(gitURL)
	if err != nil {
		return "", "", nil, nil, errors.Wrapf(err, "failed to parse gitter URL %s", gitURL)
	}

	username := provider.CurrentUsername()
	originalOrg := originalInfo.Organisation
	originalRepo := originalInfo.Name
	originRemote := "origin"
	upstreamRemote := originRemote

	// lets check if we need to fork the repository...
	fork := false
	if originalOrg != username && username != "" && originalOrg != "" && provider.ShouldForkForPullRequest(originalOrg, originalRepo, username) {
		fork = true
	}

	// Check if we are working with an existing dir
	dirExists, err := util.DirExists(dir)
	if err != nil {
		return "", "", nil, nil, errors.Wrapf(err, "failed to check if directory %s dirExists", dir)
	}
	dirIsGitRepo := false
	if dirExists {
		d, _, err := gitter.FindGitConfigDir(dir)
		if err != nil {
			return "", "", nil, nil, errors.Wrapf(err, "checking if %s is already a git repository", dir)
		}
		dirIsGitRepo = dir == d
	}

	if baseRef == "" {
		baseRef = "master"
	}

	// It's important to go via the provider to get the info and urls for cloning so that we resolve the URL to a clone
	// url. This is also useful for tests
	upstreamInfo, err := provider.GetRepository(originalOrg, originalRepo)
	if err != nil {
		return "", "", nil, nil, errors.Wrapf(err, "getting repository %s/%s", originalOrg, originalRepo)
	}
	originURL := upstreamInfo.CloneURL
	if forkName == "" {
		forkName = upstreamInfo.Name
	}

	var forkInfo *GitRepository
	// Create or use a fork on the git provider if needed
	if fork {
		forkInfo, err = provider.GetRepository(username, forkName)
		if err != nil {
			log.Logger().Debugf(errors.Wrapf(err, "getting repository %s/%s", username, forkName).Error())
			// lets try create a fork as it probably doesn't exist- using a blank organisation to force a user specific fork
			forkInfo, err = provider.ForkRepository(originalOrg, originalRepo, "")
			if err != nil {
				return "", "", nil, nil, errors.Wrapf(err, "failed to fork GitHub repo %s/%s to user %s", originalOrg, originalRepo, username)
			}
			if forkName != "" {
				renamedInfo, err := provider.RenameRepository(forkInfo.Organisation, forkInfo.Name, forkName)
				if err != nil {
					return "", "", nil, nil, errors.Wrapf(err, "failed to rename fork %s/%s to %s/%s", forkInfo.Organisation, forkInfo.Name, renamedInfo.Organisation, renamedInfo.Name)
				}
				forkInfo = renamedInfo
			}
			log.Logger().Infof("Forked Git repository to %s\n", util.ColorInfo(forkInfo.HTMLURL))
		}
		originURL = forkInfo.CloneURL
	}

	// Prepare the git repo
	if !dirExists {
		// If the directory doesn't already exist, create it
		err := os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return "", "", nil, nil, fmt.Errorf("failed to create directory %s due to %s", dir, err)
		}
	}

	stashed := false
	if !dirIsGitRepo {
		err = gitter.Init(dir)
		if err != nil {
			return "", "", nil, nil, errors.Wrapf(err, "failed to run git init in %s", dir)
		}
		// Need an initial commit to make branching work. We'll do an empty commit
		err = gitter.AddCommit(dir, "initial commit")
		if err != nil {
			return "", "", nil, nil, errors.WithStack(err)
		}
	} else {
		err = gitter.StashPush(dir)
		stashed = true
		if err != nil {
			return "", "", nil, nil, errors.WithStack(err)
		}
	}

	// The long form of "git clone" has the advantage of working fine on an existing git repo and avoids checking out master
	// and then another branch
	err = gitter.SetRemoteURL(dir, originRemote, originURL)
	if err != nil {
		return "", "", nil, nil, errors.Wrapf(err, "failed to set %s url to %s", originRemote, originURL)
	}
	if fork {
		upstreamRemote = "upstream"
		err := gitter.SetRemoteURL(dir, upstreamRemote, upstreamInfo.CloneURL)
		if err != nil {
			return "", "", nil, nil, errors.Wrapf(err, "setting remote upstream %q in forked environment repo", gitURL)
		}
	}

	userDetails := provider.UserAuth()
	originFetchURL, err := gitter.CreateAuthenticatedURL(originURL, &userDetails)
	if err != nil {
		return "", "", nil, nil, errors.Wrapf(err, "failed to create authenticated fetch URL for %s", originURL)
	}
	err = gitter.FetchBranch(dir, originFetchURL, fmt.Sprintf("%s:remotes/%s/%s", branchName, originRemote, branchName))

	if err != nil && !IsCouldntFindRemoteRefError(err, branchName) { // We can safely ignore missing remote branches, as they just don't exist
		return "", "", nil, nil, errors.Wrapf(err, "fetching %s %s", originRemote, branchName)
	}

	if upstreamRemote != originRemote || baseRef != branchName {
		upstreamFetchURL, err := gitter.CreateAuthenticatedURL(upstreamInfo.CloneURL, &userDetails)
		if err != nil {
			return "", "", nil, nil, errors.Wrapf(err, "failed to create authenticated fetch URL for %s", upstreamInfo.CloneURL)
		}
		// We're going to start our work from baseRef on the upstream
		err = gitter.FetchBranch(dir, upstreamFetchURL, fmt.Sprintf("%s:remotes/%s/%s", baseRef, upstreamRemote, baseRef))
		if err != nil {
			return "", "", nil, nil, errors.WithStack(err)
		}
	}

	// Work out what branch to use and check it out
	branchExists, err := remoteBranchExists(baseRef, branchName, dir, gitter, originRemote)
	if err != nil {
		return "", "", nil, nil, errors.WithStack(err)
	}
	localBranchName := gitter.ConvertToValidBranchName(branchName)
	localBranches, err := gitter.LocalBranches(dir)
	if err != nil {
		return "", "", nil, nil, errors.WithStack(err)
	}
	localBranchExists := util.Contains(localBranches, localBranchName)

	// We always want to make sure our local branch is in the right state, whether we created it or checked it out
	resetish := baseRef
	remoteBaseRefExists, err := remoteBranchExists(baseRef, "", dir, gitter, upstreamRemote)
	if err != nil {
		return "", "", nil, nil, errors.WithStack(err)
	}
	if remoteBaseRefExists {
		resetish = fmt.Sprintf("%s/%s", upstreamRemote, baseRef)
	}

	var toCherryPick []GitCommit
	if !localBranchExists {
		err = gitter.CreateBranch(dir, localBranchName)
		if err != nil {
			return "", "", nil, nil, errors.WithStack(err)
		}
	} else if dirExists {
		toCherryPick, err = gitter.GetCommitsNotOnAnyRemote(dir, localBranchName)
		if err != nil {
			return "", "", nil, nil, errors.WithStack(err)
		}
	}

	err = gitter.Checkout(dir, localBranchName)
	if err != nil {
		return "", "", nil, nil, errors.Wrapf(err, "failed to run git checkout %s", localBranchName)
	}
	err = gitter.Reset(dir, resetish, true)
	if err != nil {
		return "", "", nil, nil, errors.Wrapf(err, "failed to run git reset --hard %s", resetish)
	}

	// Merge in any local committed changes we found
	if len(toCherryPick) > 0 {
		log.Logger().Infof("Attempting to cherry pick commits that were on %s but not yet pushed", localBranchName)
	}
	for _, c := range toCherryPick {
		err = gitter.CherryPick(dir, c.SHA)
		if err != nil {
			if IsEmptyCommitError(err) {
				log.Logger().Debugf("  Ignoring %s as is empty", c.OneLine())
				err = gitter.Reset(dir, "", true)
				if err != nil {
					return "", "", nil, nil, errors.Wrapf(err, "running git reset --hard")
				}
			} else {
				log.Logger().Warnf(errors.Wrapf(err, "Unable to cherry-pick %s", util.ColorWarning(c.SHA)).Error())
			}
		} else {
			log.Logger().Infof("  Cherry-picked %s", c.OneLine())
		}
	}
	if len(toCherryPick) > 0 {
		log.Logger().Infof("")
	}

	// one possibility is that the baseRef is not in the same state as an already existing branch, so let's merge in the
	// already existing branch from the users fork. This is idempotent (safe if that's already the state)
	// Only do this if the remote branch exists
	if branchExists {
		err = gitter.Merge(dir, fmt.Sprintf("%s/%s", originRemote, branchName))
		if err != nil {
			return "", "", nil, nil, errors.WithStack(err)
		}
	}

	if stashed {
		err = gitter.StashPop(dir)
		if err != nil && !IsNoStashEntriesError(err) { // Ignore no stashes as that's just because there was nothing to stash
			return "", "", nil, nil, errors.Wrapf(err, "unable to pop the stash")
		}
	}

	return dir, baseRef, upstreamInfo, forkInfo, nil
}

// A PullRequestFilter defines a filter for finding pull requests
type PullRequestFilter struct {
	Labels []string
	Number *int
}

func (f *PullRequestFilter) String() string {
	if f.Number != nil {
		return fmt.Sprintf("Pull Request #%d", f.Number)
	}
	return strings.Join(f.Labels, ", ")
}

// FilterOpenPullRequests looks for any pull requests on the owner/repo where all the labels match
func FilterOpenPullRequests(provider GitProvider, owner string, repo string, filter PullRequestFilter) ([]*GitPullRequest, error) {
	openPRs, err := provider.ListOpenPullRequests(owner, repo)
	if err != nil {
		return nil, errors.Wrapf(err, "listing open pull requests on %s/%s", owner, repo)
	}
	answer := make([]*GitPullRequest, 0)
	for _, pr := range openPRs {
		if len(filter.Labels) > 0 {
			found := 0
			for _, label := range filter.Labels {
				f := false
				for _, prLabel := range pr.Labels {
					if label == util.DereferenceString(prLabel.Name) {
						f = true
					}
				}
				if f {
					found++
				}
			}
			if len(filter.Labels) == found {
				answer = append(answer, pr)
			}
		}
		if filter.Number != nil && filter.Number == pr.Number {
			answer = append(answer, pr)
		}
	}
	return answer, nil
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

		return fmt.Sprintf("%s-%s", validBranchName, branchNameUUID.String()), nil
	}
	return validBranchName, nil
}

func remoteBranchExists(baseRef string, branchName string, dir string, gitter Gitter, remoteName string) (bool, error) {

	if branchName == "" {
		branchName = baseRef
	}
	validBranchName := gitter.ConvertToValidBranchName(branchName)

	branchNames, err := gitter.RemoteBranchNames(dir, fmt.Sprintf("remotes/%s/", remoteName))
	if err != nil {
		return false, errors.Wrapf(err, "Failed to load remote branch names")
	}
	if util.StringArrayIndex(branchNames, validBranchName) >= 0 {
		if err != nil {
			return false, errors.WithStack(err)
		}
		return true, nil
	}
	return false, nil
}

//IsUnadvertisedObjectError returns true if the reason for the error is that the request was for an object that is unadvertised (i.e. doesn't exist)
func IsUnadvertisedObjectError(err error) bool {
	return strings.Contains(err.Error(), "Server does not allow request for unadvertised object")
}

func parseAuthor(l string) (string, string) {
	open := strings.Index(l, "<")
	close := strings.Index(l, ">")
	return strings.TrimSpace(l[:open]), strings.TrimSpace(l[open+1 : close])
}

// IsCouldntFindRemoteRefError returns true if the error is due to the remote ref not being found
func IsCouldntFindRemoteRefError(err error, ref string) bool {
	return strings.Contains(strings.ToLower(err.Error()), fmt.Sprintf("couldn't find remote ref %s", ref))
}

// IsCouldNotPopTheStashError returns true if the error is due to the stash not being able to be popped, often because
// whatever is in the stash cannot be applied to the new state
func IsCouldNotPopTheStashError(err error) bool {
	return strings.Contains(err.Error(), "could not pop the stash")
}

// IsNoStashEntriesError returns true if the error is due to no stash entries found
func IsNoStashEntriesError(err error) bool {
	return strings.Contains(err.Error(), "No stash entries found.")
}

// FindTagForVersion will find a tag for a version number (first fetching the tags, then looking for a tag <version>
// then trying the common convention v<version>). It will return the tag or an error if the tag can't be found.
func FindTagForVersion(dir string, version string, gitter Gitter) (string, error) {
	err := gitter.FetchTags(dir)
	if err != nil {
		return "", errors.Wrapf(err, "fetching tags for %s", dir)
	}
	answer := ""
	tags, err := gitter.FilterTags(dir, version)
	if err != nil {
		return "", errors.Wrapf(err, "listing tags for %s", version)
	}
	if len(tags) == 1 {
		answer = tags[0]
	} else if len(tags) == 0 {
		// try with v
		filter := fmt.Sprintf("v%s", version)
		tags, err := gitter.FilterTags(dir, filter)
		if err != nil {
			return "", errors.Wrapf(err, "listing tags for %s", filter)
		}
		if len(tags) == 1 {
			answer = tags[0]
		} else {
			return "", errors.Errorf("cannot resolve %s to a single git object (searching for tag %s and tag %s), found %+v", version, version, filter, tags)
		}
	} else {
		return "", errors.Errorf("cannot resolve %s to a single git object, found %+v", version, tags)
	}
	return answer, nil
}

//DuplicateGitRepoFromCommitish will duplicate branches (but not tags) from fromGitURL to toOrg/toName. It will reset the
// head of the toBranch on the duplicated repo to fromCommitish. It returns the GitRepository for the duplicated repo
func DuplicateGitRepoFromCommitish(toOrg string, toName string, fromGitURL string, fromCommitish string, toBranch string, private bool, provider GitProvider, gitter Gitter) (*GitRepository, error) {
	duplicateInfo, err := provider.GetRepository(toOrg, toName)
	// If the duplicate doesn't exist create it
	if err != nil {
		log.Logger().Debugf(errors.Wrapf(err, "getting repository %s/%s", toOrg, toName).Error())
		fromInfo, err := ParseGitURL(fromGitURL)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing %s", fromGitURL)
		}
		fromInfo, err = provider.GetRepository(fromInfo.Organisation, fromInfo.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "getting repo for %s", fromGitURL)
		}
		duplicateInfo, err = provider.CreateRepository(toOrg, toName, private)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create GitHub repo %s/%s", toOrg, toName)
		}
		dir, err := ioutil.TempDir("", "")
		if err != nil {
			return nil, errors.WithStack(err)
		}
		err = gitter.Clone(fromInfo.CloneURL, dir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to clone %s", fromInfo.CloneURL)
		}
		if !strings.Contains(fromCommitish, "/") {
			// if the commitish looks like a tag, fetch the tags
			err = gitter.FetchTags(dir)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to fetch tags fromGitURL %s", fromInfo.CloneURL)
			}
		} else {
			parts := strings.Split(fromCommitish, "/")
			err = gitter.FetchBranch(dir, parts[0], parts[1])
			if err != nil {
				return nil, errors.Wrapf(err, "failed to fetch %s fromGitURL %s", fromCommitish, fromInfo.CloneURL)
			}
		}
		err = gitter.Reset(dir, fromCommitish, true)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to reset to %s", fromCommitish)
		}
		userDetails := provider.UserAuth()
		duplicatePushURL, err := gitter.CreateAuthenticatedURL(duplicateInfo.CloneURL, &userDetails)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create push URL for %s", duplicateInfo.CloneURL)
		}
		err = gitter.SetRemoteURL(dir, "origin", duplicateInfo.CloneURL)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to set remote url to %s", duplicateInfo.CloneURL)
		}
		err = gitter.Push(dir, duplicatePushURL, true, false, fmt.Sprintf("%s:%s", "HEAD", toBranch))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to push HEAD to %s", toBranch)
		}
		log.Logger().Infof("Duplicated Git repository %s to %s\n", util.ColorInfo((fromInfo.HTMLURL)), util.ColorInfo(duplicateInfo.HTMLURL))
		log.Logger().Infof("Setting upstream to %s\n", util.ColorInfo(duplicateInfo.HTMLURL))
	}
	return duplicateInfo, nil
}

// GetGitInfoFromDirectory obtains remote origin HTTPS and current branch of a given directory and fails if it's not a git repository
func GetGitInfoFromDirectory(dir string, gitter Gitter) (string, string, error) {
	_, gitConfig, err := gitter.FindGitConfigDir(dir)
	if err != nil {
		return "", "", errors.Wrapf(err, "there was a problem obtaining the git config dir of directory %s", dir)
	}
	remoteGitURL, err := gitter.DiscoverRemoteGitURL(gitConfig)
	if err != nil {
		return "", "", errors.Wrapf(err, "there was a problem obtaining the remote Git URL of directory %s", dir)
	}
	currentBranch, err := gitter.Branch(dir)
	if err != nil {
		return "", "", errors.Wrapf(err, "there was a problem obtaining the current branch on directory %s", dir)
	}
	g, err := ParseGitURL(remoteGitURL)
	if err != nil {
		return "", "", errors.Wrapf(err, "there was a problem parsing the Git URL %s to HTTPS", remoteGitURL)
	}

	return g.HttpsURL(), currentBranch, nil
}
