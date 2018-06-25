package gits

import (
	"io"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

type Gitter interface {
	FindGitConfigDir(dir string) (string, string, error)
	ToGitLabels(names []string) []GitLabel
	PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer)

	GitStatus(dir string) error
	GetGitServer(dir string) (string, error)
	GetGitInfo(dir string) (*GitRepositoryInfo, error)
	GitIsFork(gitProvider GitProvider, gitInfo *GitRepositoryInfo, dir string) (bool, error)
	GitVersion() (string, error)
	GitRepoName(org, repoName string) string

	GitUsername(dir string) (string, error)
	GitSetUsername(dir string, username string) error
	GitEmail(dir string) (string, error)
	GitSetEmail(dir string, email string) error
	GetAuthorEmailForCommit(dir string, sha string) (string, error)

	GitInit(dir string) error
	GitClone(url string, directory string) error
	GitPush(dir string) error
	GitPushMaster(dir string) error
	GitPushTag(dir string, tag string) error
	GitCreatePushURL(cloneURL string, userAuth *auth.UserAuth) (string, error)
	GitForcePushBranch(dir string, localBranch string, remoteBranch string) error
	GitCloneOrPull(url string, directory string) error
	GitPull(dir string) error
	GitPullUpstream(dir string) error

	GitAddRemote(dir string, url string, remote string) error
	SetRemoteURL(dir string, name string, gitURL string) error
	GitUpdateRemote(dir, url string) error
	DiscoverRemoteGitURL(gitConf string) (string, error)
	DiscoverUpstreamGitURL(gitConf string) (string, error)
	GitGetRemoteBranches(dir string) ([]string, error)
	GitGetRemoteBranchNames(dir string, prefix string) ([]string, error)
	GetRemoteUrl(config *gitcfg.Config, name string) string

	GitGetBranch(dir string) (string, error)
	GitCreateBranch(dir string, branch string) error
	CheckoutRemoteBranch(dir string, branch string) error
	GitCheckout(dir string, branch string) error
	ConvertToValidBranchName(name string) string

	GitStash(dir string) error

	GitRemove(dir, fileName string) error
	GitAdd(dir string, args ...string) error

	GitCommitIfChanges(dir string, message string) error
	GitCommitDir(dir string, message string) error
	GitAddCommmit(dir string, msg string) error
	HasChanges(dir string) (bool, error)

	GetPreviousGitTagSHA(dir string) (string, error)
	GetCurrentGitTagSHA(dir string) (string, error)
	GitFetchTags(dir string) error
	GitTags(dir string) ([]string, error)
	GitCreateTag(dir string, tag string, msg string) error

	GetRevisionBeforeDate(dir string, t time.Time) (string, error)
	GetRevisionBeforeDateText(dir string, dateText string) (string, error)
}
