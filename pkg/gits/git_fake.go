package gits

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

type GitRemote struct {
	Name string
	URL  string
}

type GitTag struct {
	Name    string
	Message string
}

type GitFake struct {
	Remotes        []GitRemote
	Branches       []string
	BranchesRemote []string
	CurrentBranch  string
	AccessTokenURL string
	RepoInfo       GitRepositoryInfo
	Fork           bool
	GitVersion     string
	UserInfo       GitUser
	Commits        []GitCommit
	Changes        bool
	GitTags        []GitTag
	Revision       string
}

func (g *GitFake) FindGitConfigDir(dir string) (string, string, error) {
	return dir, dir, nil
}

func (g *GitFake) ToGitLabels(names []string) []GitLabel {
	labels := []GitLabel{}
	for _, n := range names {
		labels = append(labels, GitLabel{Name: n})
	}
	return labels
}

func (g *GitFake) PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer) {
	fmt.Fprintf(o, "Access token URL: %s\n\n", g.AccessTokenURL)
}

func (g *GitFake) Status(dir string) error {
	return nil
}

func (g *GitFake) Server(dir string) (string, error) {
	return g.RepoInfo.HostURL(), nil
}

func (g *GitFake) Info(dir string) (*GitRepositoryInfo, error) {
	return &g.RepoInfo, nil
}

func (g *GitFake) IsFork(gitProvider GitProvider, gitInfo *GitRepositoryInfo, dir string) (bool, error) {
	return g.Fork, nil
}

func (g *GitFake) Version() (string, error) {
	return g.GitVersion, nil
}

func (g *GitFake) RepoName(org string, repoName string) string {
	if org != "" {
		return org + "/" + repoName
	}
	return repoName
}

func (g *GitFake) Username(dir string) (string, error) {
	return g.UserInfo.Name, nil
}

func (g *GitFake) SetUsername(dir string, username string) error {
	g.UserInfo.Name = username
	return nil
}

func (g *GitFake) Email(dir string) (string, error) {
	return g.UserInfo.Email, nil
}

func (g *GitFake) SetEmail(dir string, email string) error {
	g.UserInfo.Email = email
	return nil
}

func (g *GitFake) GetAuthorEmailForCommit(dir string, sha string) (string, error) {
	for _, commit := range g.Commits {
		if commit.SHA == sha {
			return commit.Author.Email, nil
		}
	}
	return "", errors.New("No commit found with given SHA")
}

func (g *GitFake) Init(dir string) error {
	return nil
}

func (g *GitFake) Clone(url string, directory string) error {
	return nil
}

func (g *GitFake) Push(dir string) error {
	return nil
}

func (g *GitFake) PushMaster(dir string) error {
	return nil
}

func (g *GitFake) PushTag(dir string, tag string) error {
	return nil
}

func (g *GitFake) CreatePushURL(cloneURL string, userAuth *auth.UserAuth) (string, error) {
	u, err := url.Parse(cloneURL)
	if err != nil {
		return cloneURL, nil
	}
	if userAuth.Username != "" || userAuth.ApiToken != "" {
		u.User = url.UserPassword(userAuth.Username, userAuth.ApiToken)
		return u.String(), nil
	}
	return cloneURL, nil
}

func (g *GitFake) ForcePushBranch(dir string, localBranch string, remoteBranch string) error {
	return nil
}

func (g *GitFake) CloneOrPull(url string, directory string) error {
	return nil
}

func (g *GitFake) Pull(dir string) error {
	return nil
}

func (g *GitFake) PullUpstream(dir string) error {
	return nil
}

func (g *GitFake) AddRemote(dir string, name string, url string) error {
	r := GitRemote{
		Name: name,
		URL:  url,
	}
	g.Remotes = append(g.Remotes, r)
	return nil
}

func (g *GitFake) findRemote(name string) (*GitRemote, error) {
	for _, remote := range g.Remotes {
		if remote.Name == name {
			return &remote, nil
		}
	}
	return nil, fmt.Errorf("no remote found with name '%s'", name)
}

func (g *GitFake) SetRemoteURL(dir string, name string, gitURL string) error {
	remote, err := g.findRemote(name)
	if err != nil {
		r := GitRemote{
			Name: name,
			URL:  gitURL,
		}
		g.Remotes = append(g.Remotes, r)
		return nil
	}
	remote.URL = gitURL
	return nil
}

func (g *GitFake) UpdateRemote(dir string, url string) error {
	return g.SetRemoteURL(dir, "origin", url)
}

func (g *GitFake) DiscoverRemoteGitURL(gitConf string) (string, error) {
	origin, err := g.findRemote("origin")
	if err != nil {
		upstream, err := g.findRemote("upstream")
		if err != nil {
			return "", err
		}
		return upstream.URL, nil
	}
	return origin.URL, nil
}

func (g *GitFake) DiscoverUpstreamGitURL(gitConf string) (string, error) {
	upstream, err := g.findRemote("upstream")
	if err != nil {
		origin, err := g.findRemote("origin")
		if err != nil {
			return "", err
		}
		return origin.URL, nil
	}
	return upstream.URL, nil
}

func (g *GitFake) RemoteBranches(dir string) ([]string, error) {
	return g.BranchesRemote, nil
}

func (g *GitFake) RemoteBranchNames(dir string, prefix string) ([]string, error) {
	remoteBranches := []string{}
	for _, remoteBranch := range g.BranchesRemote {
		remoteBranch = strings.TrimSpace(strings.TrimPrefix(remoteBranch, "* "))
		if prefix != "" {
			remoteBranch = strings.TrimPrefix(remoteBranch, prefix)
		}
		remoteBranches = append(remoteBranches, remoteBranch)
	}
	return remoteBranches, nil
}

func (g *GitFake) GetRemoteUrl(config *gitcfg.Config, name string) string {
	if len(g.Remotes) == 0 {
		return ""
	}
	return g.Remotes[0].URL
}

func (g *GitFake) Branch(dir string) (string, error) {
	return g.CurrentBranch, nil
}

func (g *GitFake) CreateBranch(dir string, branch string) error {
	g.Branches = append(g.Branches, branch)
	return nil
}

func (g *GitFake) CheckoutRemoteBranch(dir string, branch string) error {
	g.CurrentBranch = branch
	g.Branches = append(g.Branches, branch)
	return nil
}

func (g *GitFake) Checkout(dir string, branch string) error {
	g.CurrentBranch = branch
	return nil
}

func (g *GitFake) ConvertToValidBranchName(name string) string {
	name = strings.TrimSuffix(name, "/")
	name = strings.TrimSuffix(name, ".lock")
	var buffer bytes.Buffer

	last := ' '
	for _, ch := range name {
		if ch <= 32 {
			ch = replaceInvalidBranchChars
		}
		switch ch {
		case '~':
			ch = replaceInvalidBranchChars
		case '^':
			ch = replaceInvalidBranchChars
		case ':':
			ch = replaceInvalidBranchChars
		case ' ':
			ch = replaceInvalidBranchChars
		case '\n':
			ch = replaceInvalidBranchChars
		case '\r':
			ch = replaceInvalidBranchChars
		case '\t':
			ch = replaceInvalidBranchChars
		}
		if ch != replaceInvalidBranchChars || last != replaceInvalidBranchChars {
			buffer.WriteString(string(ch))
		}
		last = ch
	}
	return buffer.String()
}

func (g *GitFake) Stash(dir string) error {
	return nil
}

func (g *GitFake) Remove(dir string, fileName string) error {
	return nil
}

func (g *GitFake) Add(dir string, args ...string) error {
	return nil
}

func (g *GitFake) CommitIfChanges(dir string, message string) error {
	commit := GitCommit{
		SHA:       "",
		Message:   message,
		Author:    &g.UserInfo,
		URL:       g.RepoInfo.URL,
		Branch:    g.CurrentBranch,
		Committer: &g.UserInfo,
	}
	g.Commits = append(g.Commits, commit)
	return nil
}

func (g *GitFake) CommitDir(dir string, message string) error {
	return g.CommitIfChanges(dir, message)
}

func (g *GitFake) AddCommmit(dir string, msg string) error {
	return g.CommitIfChanges(dir, msg)
}

func (g *GitFake) HasChanges(dir string) (bool, error) {
	return g.Changes, nil
}

func (g *GitFake) GetPreviousGitTagSHA(dir string) (string, error) {
	len := len(g.Commits)
	if len < 2 {
		return "", errors.New("no previous commit found")
	}
	return g.Commits[len-2].SHA, nil
}

func (g *GitFake) GetCurrentGitTagSHA(dir string) (string, error) {
	len := len(g.Commits)
	if len < 1 {
		return "", errors.New("no previous commit found")
	}
	return g.Commits[len-1].SHA, nil
}

func (g *GitFake) FetchTags(dir string) error {
	return nil
}

func (g *GitFake) Tags(dir string) ([]string, error) {
	tags := []string{}
	for _, tag := range g.GitTags {
		tags = append(tags, tag.Name)
	}
	return tags, nil
}

func (g *GitFake) CreateTag(dir string, tag string, msg string) error {
	t := GitTag{
		Name:    tag,
		Message: msg,
	}
	g.GitTags = append(g.GitTags, t)
	return nil
}

func (g *GitFake) GetRevisionBeforeDate(dir string, t time.Time) (string, error) {
	return g.Revision, nil
}

func (g *GitFake) GetRevisionBeforeDateText(dir string, dateText string) (string, error) {
	return g.Revision, nil
}
