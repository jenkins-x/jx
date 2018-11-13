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

func (g *GitFake) ListOrganisations() ([]GitOrganisation, error) {
	answer := []GitOrganisation{}
	for _, org := range g.Organisations {
		answer = append(answer, org.Organisation)
	}
	return answer, nil
}

func (g *GitFake) ListRepositories(org string) ([]*GitRepository, error) {
	organisation := g.Organisations[org]
	if organisation == nil {
		return nil, nil
	}
	return organisation.Repositories, nil
}

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

func (g *GitFake) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	panic("implement me")
}

func (g *GitFake) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	panic("implement me")
}

func (g *GitFake) ValidateRepositoryName(org string, name string) error {
	panic("implement me")
}

func (g *GitFake) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	panic("implement me")
}

func (g *GitFake) UpdatePullRequestStatus(pr *GitPullRequest) error {
	panic("implement me")
}

func (g *GitFake) GetPullRequest(owner string, repo *GitRepositoryInfo, number int) (*GitPullRequest, error) {
	panic("implement me")
}

func (g *GitFake) GetPullRequestCommits(owner string, repo *GitRepositoryInfo, number int) ([]*GitCommit, error) {
	panic("implement me")
}

func (g *GitFake) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	panic("implement me")
}

func (g *GitFake) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	panic("implement me")
}

func (g *GitFake) UpdateCommitStatus(org string, repo string, sha string, status *GitRepoStatus) (*GitRepoStatus, error) {
	panic("implement me")
}

func (g *GitFake) MergePullRequest(pr *GitPullRequest, message string) error {
	panic("implement me")
}

func (g *GitFake) CreateWebHook(data *GitWebHookArguments) error {
	log.Infof("Created fake WebHook for %#v\n", data)
	g.WebHooks = append(g.WebHooks, data)
	return nil
}

func (g *GitFake) ListWebHooks(org string, repo string) ([]*GitWebHookArguments, error) {
	return g.WebHooks, nil
}

func (g *GitFake) UpdateWebHook(data *GitWebHookArguments) error {
	for idx, wh := range g.WebHooks {
		if wh.URL == data.URL || wh.ID == data.ID {
			g.WebHooks[idx] = data
		}
	}
	return nil
}

func (g *GitFake) IsGitHub() bool {
	return false
}

func (g *GitFake) IsGitea() bool {
	return false
}

func (g *GitFake) IsBitbucketCloud() bool {
	return false
}

func (g *GitFake) IsBitbucketServer() bool {
	return false
}

func (g *GitFake) IsGerrit() bool {
	return false
}

func (g *GitFake) Kind() string {
	return KindGitFake
}

func (g *GitFake) GetIssue(org string, name string, number int) (*GitIssue, error) {
	panic("implement me")
}

func (g *GitFake) IssueURL(org string, name string, number int, isPull bool) string {
	panic("implement me")
}

func (g *GitFake) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {
	panic("implement me")
}

func (g *GitFake) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	panic("implement me")
}

func (g *GitFake) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	panic("implement me")
}

func (g *GitFake) HasIssues() bool {
	panic("implement me")
}

func (g *GitFake) AddPRComment(pr *GitPullRequest, comment string) error {
	panic("implement me")
}

func (g *GitFake) CreateIssueComment(owner string, repo string, number int, comment string) error {
	panic("implement me")
}

func (g *GitFake) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	panic("implement me")
}

func (g *GitFake) ListReleases(org string, name string) ([]*GitRelease, error) {
	panic("implement me")
}

func (g *GitFake) GetContent(org string, name string, path string, ref string) (*GitFileContent, error) {
	panic("implement me")
}

func (g *GitFake) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/fake-webhook/"
}

func (g *GitFake) Label() string {
	return "fake"
}

func (g *GitFake) ServerURL() string {
	return g.serverURL
}

func (g *GitFake) BranchArchiveURL(org string, name string, branch string) string {
	panic("implement me")
}

func (g *GitFake) CurrentUsername() string {
	return g.GitUser.Login
}

func (g *GitFake) UserAuth() auth.UserAuth {
	return g.userAuth
}

func (g *GitFake) UserInfo(username string) *GitUser {
	panic("implement me")
}

func (g *GitFake) AddCollaborator(string, string, string) error {
	panic("implement me")
}

func (g *GitFake) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	panic("implement me")
}

func (g *GitFake) AcceptInvitation(int64) (*github.Response, error) {
	panic("implement me")
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
	return g.GitUser.Name, nil
}

func (g *GitFake) SetUsername(dir string, username string) error {
	g.GitUser.Name = username
	return nil
}

func (g *GitFake) Email(dir string) (string, error) {
	return g.GitUser.Email, nil
}

func (g *GitFake) SetEmail(dir string, email string) error {
	g.GitUser.Email = email
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

func (g *GitFake) ShallowCloneBranch(url string, branch string, directory string) error {
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

func (g *GitFake) PullRemoteBranches(dir string) error {
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

func (g *GitFake) DeleteRemoteBranch(dir string, remoteName string, branch string) error {
	return nil
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

func (g *GitFake) CheckoutOrphan(dir string, branch string) error {
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

func (g *GitFake) FetchBranch(dir string, repo string, refspec string) error {
	return nil
}

func (g *GitFake) Stash(dir string) error {
	return nil
}

func (g *GitFake) Remove(dir string, fileName string) error {
	return nil
}

func (g *GitFake) RemoveForce(dir string, fileName string) error {
	return nil
}

func (g *GitFake) CleanForce(dir string, fileName string) error {
	return nil
}

func (g *GitFake) Add(dir string, args ...string) error {
	return nil
}

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

func (g *GitFake) Diff(dir string) (string, error) {
	return "", nil
}

func (g *GitFake) notFound() error {
	return fmt.Errorf("Not found")
}
