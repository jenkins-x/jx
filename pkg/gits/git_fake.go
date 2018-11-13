package gits

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/log"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

// GitRemote a remote url
type GitRemote struct {
	Name string
	URL  string
}

// GitTag git tag
type GitTag struct {
	Name    string
	Message string
}

// GitFake provides a fake git provider
type GitFake struct {
	Remotes        []GitRemote
	Branches       []string
	BranchesRemote []string
	CurrentBranch  string
	AccessTokenURL string
	RepoInfo       GitRepositoryInfo
	Fork           bool
	GitVersion     string
	GitUser        GitUser
	Commits        []GitCommit
	Changes        bool
	GitTags        []GitTag
	Revision       string
	serverURL      string
	userAuth       auth.UserAuth

	Organisations map[string]*FakeOrganisation
	WebHooks      []*GitWebHookArguments
}

// FakeOrganisation a fake organisation
type FakeOrganisation struct {
	Organisation GitOrganisation
	Repositories []*GitRepository
}

// NewFakeGit creates a new fake git provider
func NewFakeGit(server *auth.AuthServer, user *auth.UserAuth, git Gitter) (GitProvider, error) {
	gitUser := GitUser{}
	if user != nil {
		gitUser.Name = user.Username
		gitUser.Login = user.Username
	}
	serverURL := FakeGitURL
	if server != nil && server.URL != "" {
		serverURL = server.URL
	}
	answer := &GitFake{
		GitUser:       gitUser,
		serverURL:     serverURL,
		Organisations: map[string]*FakeOrganisation{},
	}
	if user != nil {
		answer.userAuth = *user
	}
	return answer, nil
}

// ListOrganisations list the organisations
func (g *GitFake) ListOrganisations() ([]GitOrganisation, error) {
	answer := []GitOrganisation{}
	for _, org := range g.Organisations {
		answer = append(answer, org.Organisation)
	}
	return answer, nil
}

// ListRepositories list the repos for an org
func (g *GitFake) ListRepositories(org string) ([]*GitRepository, error) {
	organisation := g.Organisations[org]
	if organisation == nil {
		return nil, nil
	}
	return organisation.Repositories, nil
}

// CreateRepository create a repo in an org
func (g *GitFake) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	organisation := g.Organisations[org]
	if organisation == nil {
		organisation := &FakeOrganisation{
			Organisation: GitOrganisation{
				Login: org,
			},
			Repositories: []*GitRepository{},
		}
		g.Organisations[org] = organisation
	}
	answer := &GitRepository{
		Name: name,
	}
	organisation.Repositories = append(organisation.Repositories, answer)
	return answer, nil
}

// GetRepository get a repo
func (g *GitFake) GetRepository(org string, name string) (*GitRepository, error) {
	organisation := g.Organisations[org]
	if organisation == nil {
		return nil, nil
	}
	for _, repo := range organisation.Repositories {
		if repo.Name == name {
			return repo, nil
		}
	}
	return nil, g.notFound()
}

// DeleteRepository delete a repo
func (g *GitFake) DeleteRepository(org string, name string) error {
	organisation := g.Organisations[org]
	if organisation == nil {
		return g.notFound()
	}
	for idx, repo := range organisation.Repositories {
		if repo.Name == name {
			organisation.Repositories = append(organisation.Repositories[0:idx], organisation.Repositories[idx+1:]...)
			return nil
		}
	}
	return g.notFound()
}

// ForkRepository fork a repo
func (g *GitFake) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	panic("implement me")
}

// RenameRepository rename a repo
func (g *GitFake) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	panic("implement me")
}

// ValidateRepositoryName validate a repo name can be used
func (g *GitFake) ValidateRepositoryName(org string, name string) error {
	panic("implement me")
}

// CreatePullRequest create a PR
func (g *GitFake) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	panic("implement me")
}

// UpdatePullRequestStatus update the status of a PR
func (g *GitFake) UpdatePullRequestStatus(pr *GitPullRequest) error {
	panic("implement me")
}

// GetPullRequest get a PR
func (g *GitFake) GetPullRequest(owner string, repo *GitRepositoryInfo, number int) (*GitPullRequest, error) {
	panic("implement me")
}

// GetPullRequestCommits get the commits for a PR
func (g *GitFake) GetPullRequestCommits(owner string, repo *GitRepositoryInfo, number int) ([]*GitCommit, error) {
	panic("implement me")
}

// PullRequestLastCommitStatus get the status of the last PR's commit
func (g *GitFake) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	panic("implement me")
}

// ListCommitStatus list the status of a commit
func (g *GitFake) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	panic("implement me")
}

// UpdateCommitStatus update the status of a commit
func (g *GitFake) UpdateCommitStatus(org string, repo string, sha string, status *GitRepoStatus) (*GitRepoStatus, error) {
	panic("implement me")
}

// MergePullRequest merge a PR
func (g *GitFake) MergePullRequest(pr *GitPullRequest, message string) error {
	panic("implement me")
}

// CreateWebHook create a webhook
func (g *GitFake) CreateWebHook(data *GitWebHookArguments) error {
	log.Infof("Created fake WebHook at %s with repo %#v\n", data.URL, data.Repo)
	g.WebHooks = append(g.WebHooks, data)
	return nil
}

// ListWebHooks list webhooks
func (g *GitFake) ListWebHooks(org string, repo string) ([]*GitWebHookArguments, error) {
	return g.WebHooks, nil
}

// UpdateWebHook update webhook details
func (g *GitFake) UpdateWebHook(data *GitWebHookArguments) error {
	repo := data.Repo
	if repo != nil {
		for idx, wh := range g.WebHooks {
			if wh.Repo != nil && wh.Repo.Organisation == repo.Organisation && wh.Repo.Name == repo.Name {
				g.WebHooks[idx] = data
			}
		}
	}
	return nil
}

// IsGitHub returns true if github
func (g *GitFake) IsGitHub() bool {
	return false
}

// IsGitHub returns true if gitea
func (g *GitFake) IsGitea() bool {
	return false
}

// IsBitbucketCloud returns true if bitbucket cloud
func (g *GitFake) IsBitbucketCloud() bool {
	return false
}

// IsBitbucketServer returns true if bitbucket server
func (g *GitFake) IsBitbucketServer() bool {
	return false
}

// IsGerrit returns true if gerrit
func (g *GitFake) IsGerrit() bool {
	return false
}

// Kind returns the kind
func (g *GitFake) Kind() string {
	return KindGitFake
}

// GetIssue get an issue
func (g *GitFake) GetIssue(org string, name string, number int) (*GitIssue, error) {
	panic("implement me")
}

// IssueURL get an issue URL
func (g *GitFake) IssueURL(org string, name string, number int, isPull bool) string {
	panic("implement me")
}

// SearchIssues search issues
func (g *GitFake) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {
	panic("implement me")
}

// SearchIssuesClosedSince search issues closed since
func (g *GitFake) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	panic("implement me")
}

// CreateIssue create an issue
func (g *GitFake) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	panic("implement me")
}

// HasIssues returns true if has issues
func (g *GitFake) HasIssues() bool {
	panic("implement me")
}

// AddPRComment add a comment to a PR
func (g *GitFake) AddPRComment(pr *GitPullRequest, comment string) error {
	panic("implement me")
}

// CreateIssueComment create a comment on an issue
func (g *GitFake) CreateIssueComment(owner string, repo string, number int, comment string) error {
	panic("implement me")
}

// UpdateRelease update a release
func (g *GitFake) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	panic("implement me")
}

// ListReleases list the releases
func (g *GitFake) ListReleases(org string, name string) ([]*GitRelease, error) {
	panic("implement me")
}

// GetContent gets the content for a file
func (g *GitFake) GetContent(org string, name string, path string, ref string) (*GitFileContent, error) {
	panic("implement me")
}

// JenkinsWebHookPath returns the path for jenkins webhooks
func (g *GitFake) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/fake-webhook/"
}

// Label return the label
func (g *GitFake) Label() string {
	return "fake"
}

// ServerURL returns the server URL
func (g *GitFake) ServerURL() string {
	return g.serverURL
}

// BranchArchiveURL returns the branch archive URL
func (g *GitFake) BranchArchiveURL(org string, name string, branch string) string {
	panic("implement me")
}

// CurrentUsername returns the current user name
func (g *GitFake) CurrentUsername() string {
	return g.GitUser.Login
}

// UserAuth returns the current user auth
func (g *GitFake) UserAuth() auth.UserAuth {
	return g.userAuth
}

// UserInfo returns the user info for the given user name
func (g *GitFake) UserInfo(username string) *GitUser {
	panic("implement me")
}

// AddCollaborator adds a collaborator
func (g *GitFake) AddCollaborator(string, string, string) error {
	panic("implement me")
}

// ListInvitations list invitations
func (g *GitFake) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	panic("implement me")
}

// AcceptInvitation accepts invitation
func (g *GitFake) AcceptInvitation(int64) (*github.Response, error) {
	panic("implement me")
}

// FindGitConfigDir finds the git config dir
func (g *GitFake) FindGitConfigDir(dir string) (string, string, error) {
	return dir, dir, nil
}

// ToGitLabels converts the labels to git labels
func (g *GitFake) ToGitLabels(names []string) []GitLabel {
	labels := []GitLabel{}
	for _, n := range names {
		labels = append(labels, GitLabel{Name: n})
	}
	return labels
}

// PrintCreateRepositoryGenerateAccessToken prints the generate access token URL
func (g *GitFake) PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer) {
	fmt.Fprintf(o, "Access token URL: %s\n\n", g.AccessTokenURL)
}

// Status check the status
func (g *GitFake) Status(dir string) error {
	return nil
}

// Server returns the server URL
func (g *GitFake) Server(dir string) (string, error) {
	return g.RepoInfo.HostURL(), nil
}

// Info returns the git repo info
func (g *GitFake) Info(dir string) (*GitRepositoryInfo, error) {
	return &g.RepoInfo, nil
}

// IsFork returns trie if this repo is a fork
func (g *GitFake) IsFork(gitProvider GitProvider, gitInfo *GitRepositoryInfo, dir string) (bool, error) {
	return g.Fork, nil
}

// Version returns the git version
func (g *GitFake) Version() (string, error) {
	return g.GitVersion, nil
}

// RepoName returns the repo name
func (g *GitFake) RepoName(org string, repoName string) string {
	if org != "" {
		return org + "/" + repoName
	}
	return repoName
}

// Username returns the current user name
func (g *GitFake) Username(dir string) (string, error) {
	return g.GitUser.Name, nil
}

// SetUsername sets the username
func (g *GitFake) SetUsername(dir string, username string) error {
	g.GitUser.Name = username
	return nil
}

// Email returns the current user git email address
func (g *GitFake) Email(dir string) (string, error) {
	return g.GitUser.Email, nil
}

// SetEmail sets the git email address
func (g *GitFake) SetEmail(dir string, email string) error {
	g.GitUser.Email = email
	return nil
}

// GetAuthorEmailForCommit returns the author email for a commit
func (g *GitFake) GetAuthorEmailForCommit(dir string, sha string) (string, error) {
	for _, commit := range g.Commits {
		if commit.SHA == sha {
			return commit.Author.Email, nil
		}
	}
	return "", errors.New("No commit found with given SHA")
}

// Init initialises git in a dir
func (g *GitFake) Init(dir string) error {
	return nil
}

// Clone clones the repo to the given dir
func (g *GitFake) Clone(url string, directory string) error {
	return nil
}

// ShallowCloneBranch shallow clone of a branch
func (g *GitFake) ShallowCloneBranch(url string, branch string, directory string) error {
	return nil
}

// Push performs a git push
func (g *GitFake) Push(dir string) error {
	return nil
}

// PushMaster pushes to master
func (g *GitFake) PushMaster(dir string) error {
	return nil
}

// PushTag pushes a tag
func (g *GitFake) PushTag(dir string, tag string) error {
	return nil
}

// CreatePushURL creates a Push URL
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

// ForcePushBranch force push a branch
func (g *GitFake) ForcePushBranch(dir string, localBranch string, remoteBranch string) error {
	return nil
}

// CloneOrPull performs a clone or pull
func (g *GitFake) CloneOrPull(url string, directory string) error {
	return nil
}

// Pull git pulls
func (g *GitFake) Pull(dir string) error {
	return nil
}

// PullRemoteBranches pull remote branches
func (g *GitFake) PullRemoteBranches(dir string) error {
	return nil
}

// PullUpstream pulls upstream
func (g *GitFake) PullUpstream(dir string) error {
	return nil
}

// AddRemote adds a remote
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

// SetRemoteURL sets a remote URL
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

// UpdateRemote updates a remote
func (g *GitFake) UpdateRemote(dir string, url string) error {
	return g.SetRemoteURL(dir, "origin", url)
}

// DeleteRemoteBranch deletes a remote branch
func (g *GitFake) DeleteRemoteBranch(dir string, remoteName string, branch string) error {
	return nil
}

// DiscoverRemoteGitURL discover the remote git URL
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

// DiscoverUpstreamGitURL discover the upstream git URL
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

// RemoteBranches list the remote branches
func (g *GitFake) RemoteBranches(dir string) ([]string, error) {
	return g.BranchesRemote, nil
}

// RemoteBranchNames list the remote branch names
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

// GetRemoteUrl get the remote URL
func (g *GitFake) GetRemoteUrl(config *gitcfg.Config, name string) string {
	if len(g.Remotes) == 0 {
		return ""
	}
	return g.Remotes[0].URL
}

// Branch returns the current branch
func (g *GitFake) Branch(dir string) (string, error) {
	return g.CurrentBranch, nil
}

// CreateBranch creates a branch
func (g *GitFake) CreateBranch(dir string, branch string) error {
	g.Branches = append(g.Branches, branch)
	return nil
}

// CheckoutRemoteBranch checkout remote branch
func (g *GitFake) CheckoutRemoteBranch(dir string, branch string) error {
	g.CurrentBranch = branch
	g.Branches = append(g.Branches, branch)
	return nil
}

// Checkout checkout the branch
func (g *GitFake) Checkout(dir string, branch string) error {
	g.CurrentBranch = branch
	return nil
}

// CheckoutOrphan checkout the orphan
func (g *GitFake) CheckoutOrphan(dir string, branch string) error {
	g.CurrentBranch = branch
	return nil
}

// ConvertToValidBranchName converts the name to a valid branch name
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

// FetchBranch fetch branch
func (g *GitFake) FetchBranch(dir string, repo string, refspec string) error {
	return nil
}

// Stash git stash
func (g *GitFake) Stash(dir string) error {
	return nil
}

// Remove a file from git
func (g *GitFake) Remove(dir string, fileName string) error {
	return nil
}

// RemoveForce remove force
func (g *GitFake) RemoveForce(dir string, fileName string) error {
	return nil
}

// CleanForce clean force
func (g *GitFake) CleanForce(dir string, fileName string) error {
	return nil
}

// Add add files to git
func (g *GitFake) Add(dir string, args ...string) error {
	return nil
}

// CommitIfChanges git commit if there are changes
func (g *GitFake) CommitIfChanges(dir string, message string) error {
	commit := GitCommit{
		SHA:       "",
		Message:   message,
		Author:    &g.GitUser,
		URL:       g.RepoInfo.URL,
		Branch:    g.CurrentBranch,
		Committer: &g.GitUser,
	}
	g.Commits = append(g.Commits, commit)
	return nil
}

// CommitDir commit a dir
func (g *GitFake) CommitDir(dir string, message string) error {
	return g.CommitIfChanges(dir, message)
}

// AddCommmit add a commit
func (g *GitFake) AddCommmit(dir string, msg string) error {
	return g.CommitIfChanges(dir, msg)
}

// HasChanges returns true if has changes in git
func (g *GitFake) HasChanges(dir string) (bool, error) {
	return g.Changes, nil
}

// GetPreviousGitTagSHA returns the previous git tag SHA
func (g *GitFake) GetPreviousGitTagSHA(dir string) (string, error) {
	len := len(g.Commits)
	if len < 2 {
		return "", errors.New("no previous commit found")
	}
	return g.Commits[len-2].SHA, nil
}

// GetCurrentGitTagSHA returns the current git tag sha
func (g *GitFake) GetCurrentGitTagSHA(dir string) (string, error) {
	len := len(g.Commits)
	if len < 1 {
		return "", errors.New("no previous commit found")
	}
	return g.Commits[len-1].SHA, nil
}

// FetchTags fetches tags
func (g *GitFake) FetchTags(dir string) error {
	return nil
}

// Tags lists the tags
func (g *GitFake) Tags(dir string) ([]string, error) {
	tags := []string{}
	for _, tag := range g.GitTags {
		tags = append(tags, tag.Name)
	}
	return tags, nil
}

// CreateTag creates a tag
func (g *GitFake) CreateTag(dir string, tag string, msg string) error {
	t := GitTag{
		Name:    tag,
		Message: msg,
	}
	g.GitTags = append(g.GitTags, t)
	return nil
}

// GetRevisionBeforeDate get the revision before the date
func (g *GitFake) GetRevisionBeforeDate(dir string, t time.Time) (string, error) {
	return g.Revision, nil
}

// GetRevisionBeforeDateText get the revision before the date text
func (g *GitFake) GetRevisionBeforeDateText(dir string, dateText string) (string, error) {
	return g.Revision, nil
}

// Diff performs a git diff
func (g *GitFake) Diff(dir string) (string, error) {
	return "", nil
}

func (g *GitFake) notFound() error {
	return fmt.Errorf("Not found")
}
