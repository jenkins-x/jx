package gits

import (
	"io"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

// Gitter defines common git actions used by Jenkins X
type Gitter interface {
	FindGitConfigDir(dir string) (string, string, error)
	ToGitLabels(names []string) []GitLabel
	PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer)

	Status(dir string) error
	Server(dir string) (string, error)
	Info(dir string) (*GitRepositoryInfo, error)
	IsFork(gitProvider GitProvider, gitInfo *GitRepositoryInfo, dir string) (bool, error)
	Version() (string, error)
	RepoName(org, repoName string) string

	Username(dir string) (string, error)
	SetUsername(dir string, username string) error
	Email(dir string) (string, error)
	SetEmail(dir string, email string) error
	GetAuthorEmailForCommit(dir string, sha string) (string, error)

	Init(dir string) error
	Clone(url string, directory string) error
	Push(dir string) error
	PushMaster(dir string) error
	PushTag(dir string, tag string) error
	CreatePushURL(cloneURL string, userAuth *auth.UserAuth) (string, error)
	ForcePushBranch(dir string, localBranch string, remoteBranch string) error
	CloneOrPull(url string, directory string) error
	Pull(dir string) error
	PullUpstream(dir string) error

	AddRemote(dir string, name string, url string) error
	SetRemoteURL(dir string, name string, gitURL string) error
	UpdateRemote(dir, url string) error
	DiscoverRemoteGitURL(gitConf string) (string, error)
	DiscoverUpstreamGitURL(gitConf string) (string, error)
	RemoteBranches(dir string) ([]string, error)
	RemoteBranchNames(dir string, prefix string) ([]string, error)
	GetRemoteUrl(config *gitcfg.Config, name string) string

	Branch(dir string) (string, error)
	CreateBranch(dir string, branch string) error
	CheckoutRemoteBranch(dir string, branch string) error
	Checkout(dir string, branch string) error
	ConvertToValidBranchName(name string) string

	Stash(dir string) error

	Remove(dir, fileName string) error
	Add(dir string, args ...string) error

	CommitIfChanges(dir string, message string) error
	CommitDir(dir string, message string) error
	AddCommmit(dir string, msg string) error
	HasChanges(dir string) (bool, error)

	GetPreviousGitTagSHA(dir string) (string, error)
	GetCurrentGitTagSHA(dir string) (string, error)
	FetchTags(dir string) error
	Tags(dir string) ([]string, error)
	CreateTag(dir string, tag string, msg string) error

	GetRevisionBeforeDate(dir string, t time.Time) (string, error)
	GetRevisionBeforeDateText(dir string, dateText string) (string, error)
}
