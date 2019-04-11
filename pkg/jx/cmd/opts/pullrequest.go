package opts

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/sirupsen/logrus"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// PullRequestDetails details to pass in to create a PullRequest if the repository is modified
type PullRequestDetails struct {
	Dir               string
	RepositoryGitURL  string
	RepositoryBranch  string
	RepositoryMessage string
	BranchNameText    string
	Title             string
	Message           string
}

// CreatePullRequest creates a Pull Request on the given repository
func (options *CommonOptions) CreatePullRequest(o *PullRequestDetails, modifyFn func() error) error {
	if o.RepositoryBranch == "" {
		o.RepositoryBranch = "master"
	}
	dir := o.Dir
	originalGitURL := o.RepositoryGitURL
	message := o.RepositoryMessage
	gitter := options.Git()
	gitInfo, err := gits.ParseGitURL(originalGitURL)
	if err != nil {
		return err
	}
	provider, err := options.GitProviderForURL(originalGitURL, message)
	if err != nil {
		return err
	}

	username := provider.CurrentUsername()
	if username == "" {
		return fmt.Errorf("no git user name found")
	}

	originalOrg := gitInfo.Organisation
	originalRepo := gitInfo.Name

	repo, err := provider.GetRepository(username, originalRepo)
	if err != nil {
		if originalOrg == username {
			return err
		}

		// lets try create a fork - using a blank organisation to force a user specific fork
		repo, err = provider.ForkRepository(originalOrg, originalRepo, "")
		if err != nil {
			return errors.Wrapf(err, "failed to fork GitHub repo %s/%s to user %s", originalOrg, originalRepo, username)
		}
		logrus.Infof("Forked %s to %s\n\n", message, util.ColorInfo(repo.HTMLURL))
	}

	err = gitter.Clone(repo.CloneURL, dir)
	if err != nil {
		return errors.Wrapf(err, "cloning the %s %q", message, repo.CloneURL)
	}
	logrus.Infof("cloned fork of %s %s to %s\n", message, util.ColorInfo(repo.HTMLURL), util.ColorInfo(dir))

	err = gitter.SetRemoteURL(dir, "upstream", originalGitURL)
	if err != nil {
		return errors.Wrapf(err, "setting remote upstream %q in forked %s", originalGitURL, message)
	}
	err = gitter.PullUpstream(dir)
	if err != nil {
		return errors.Wrapf(err, "pulling upstream of forked %s", message)
	}

	branchName := gitter.ConvertToValidBranchName(o.BranchNameText)
	branchNames, err := gitter.RemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return errors.Wrapf(err, "failed to load remote branch names")
	}
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchName += "-" + string(uuid.NewUUID())
	}

	err = gitter.CreateBranch(dir, branchName)
	if err != nil {
		return err
	}
	err = gitter.Checkout(dir, branchName)
	if err != nil {
		return err
	}

	err = modifyFn()
	if err != nil {
		return err
	}

	err = gitter.Add(dir, "*", "*/*")
	if err != nil {
		return err
	}
	changes, err := gitter.HasChanges(dir)
	if err != nil {
		return err
	}
	if !changes {
		logrus.Infof("No source changes so not generating a Pull Request\n")
		return nil
	}

	err = gitter.CommitDir(dir, o.Title)
	if err != nil {
		return err
	}

	// lets find a previous PR so we can force push to its branch
	prs, err := provider.ListOpenPullRequests(gitInfo.Organisation, gitInfo.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to list open pull requests on %s", gitInfo.HTMLURL)
	}
	for _, pr := range prs {
		author := pr.Author
		if pr.Title == o.Title && author != nil && author.Login == username {
			logrus.Infof("found existing PullRequest: %s\n", util.ColorInfo(pr.URL))

			head := pr.HeadRef
			if head == nil {
				logrus.Warnf("No head value!\n")
			} else {
				headText := *head
				remoteBranch := headText
				paths := strings.SplitN(headText, ":", 2)
				if len(paths) > 1 {
					remoteBranch = paths[1]
				}
				logrus.Infof("force pushing to remote branch %s\n", util.ColorInfo(remoteBranch))
				err := gitter.ForcePushBranch(dir, branchName, remoteBranch)
				if err != nil {
					return errors.Wrapf(err, "failed to force push to remote branch %s", remoteBranch)
				}

				pr.Body = o.Message

				logrus.Infof("force pushed new pull request change to: %s\n", util.ColorInfo(pr.URL))

				err = provider.AddPRComment(pr, o.Message)
				if err != nil {
					return errors.Wrapf(err, "failed to add message to PR %s", pr.URL)
				}
				return nil
			}
		}
	}

	err = gitter.Push(dir)
	if err != nil {
		return errors.Wrapf(err, "pushing to %s in dir %q", message, dir)
	}

	base := o.RepositoryBranch

	gha := &gits.GitPullRequestArguments{
		GitRepository: gitInfo,
		Title:         o.Title,
		Body:          o.Message,
		Base:          base,
		Head:          username + ":" + branchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
		return err
	}
	logrus.Infof("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return nil
}
