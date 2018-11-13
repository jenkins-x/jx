package gits

import (
	"fmt"
	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/log"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
)

// GitFakeProvider provides a fake git provider
type GitFakeProvider struct {
	GitUser       GitUser
	Organisations map[string]*FakeOrganisation
	WebHooks      []*GitWebHookArguments

	serverURL string
	userAuth  auth.UserAuth
	git       Gitter
}

// FakeOrganisation a fake organisation
type FakeOrganisation struct {
	Organisation GitOrganisation
	Repositories []*GitRepository
}

// NewFakeGitProvider creates a new fake git provider
func NewFakeGitProvider(server *auth.AuthServer, user *auth.UserAuth, git Gitter) (GitProvider, error) {
	gitUser := GitUser{}
	if user != nil {
		gitUser.Name = user.Username
		gitUser.Login = user.Username
	}
	serverURL := FakeGitURL
	if server != nil && server.URL != "" {
		serverURL = server.URL
	}
	answer := &GitFakeProvider{
		GitUser:       gitUser,
		serverURL:     serverURL,
		Organisations: map[string]*FakeOrganisation{},
		git:           git,
	}
	if user != nil {
		answer.userAuth = *user
	}
	return answer, nil
}

// ListOrganisations list the organisations
func (g *GitFakeProvider) ListOrganisations() ([]GitOrganisation, error) {
	answer := []GitOrganisation{}
	for _, org := range g.Organisations {
		answer = append(answer, org.Organisation)
	}
	return answer, nil
}

// ListRepositories list the repos for an org
func (g *GitFakeProvider) ListRepositories(org string) ([]*GitRepository, error) {
	organisation := g.Organisations[org]
	if organisation == nil {
		return nil, nil
	}
	return organisation.Repositories, nil
}

// CreateRepository create a repo in an org
func (g *GitFakeProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
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
func (g *GitFakeProvider) GetRepository(org string, name string) (*GitRepository, error) {
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
func (g *GitFakeProvider) DeleteRepository(org string, name string) error {
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
func (g *GitFakeProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	panic("implement me")
}

// RenameRepository rename a repo
func (g *GitFakeProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	panic("implement me")
}

// ValidateRepositoryName validate a repo name can be used
func (g *GitFakeProvider) ValidateRepositoryName(org string, name string) error {
	panic("implement me")
}

// CreatePullRequest create a PR
func (g *GitFakeProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	panic("implement me")
}

// UpdatePullRequestStatus update the status of a PR
func (g *GitFakeProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
	panic("implement me")
}

// GetPullRequest get a PR
func (g *GitFakeProvider) GetPullRequest(owner string, repo *GitRepositoryInfo, number int) (*GitPullRequest, error) {
	panic("implement me")
}

// GetPullRequestCommits get the commits for a PR
func (g *GitFakeProvider) GetPullRequestCommits(owner string, repo *GitRepositoryInfo, number int) ([]*GitCommit, error) {
	panic("implement me")
}

// PullRequestLastCommitStatus get the status of the last PR's commit
func (g *GitFakeProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	panic("implement me")
}

// ListCommitStatus list the status of a commit
func (g *GitFakeProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	panic("implement me")
}

// UpdateCommitStatus update the status of a commit
func (g *GitFakeProvider) UpdateCommitStatus(org string, repo string, sha string, status *GitRepoStatus) (*GitRepoStatus, error) {
	panic("implement me")
}

// MergePullRequest merge a PR
func (g *GitFakeProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	panic("implement me")
}

// CreateWebHook create a webhook
func (g *GitFakeProvider) CreateWebHook(data *GitWebHookArguments) error {
	log.Infof("Created fake WebHook at %s with repo %#v\n", data.URL, data.Repo)
	g.WebHooks = append(g.WebHooks, data)
	return nil
}

// ListWebHooks list webhooks
func (g *GitFakeProvider) ListWebHooks(org string, repo string) ([]*GitWebHookArguments, error) {
	return g.WebHooks, nil
}

// UpdateWebHook update webhook details
func (g *GitFakeProvider) UpdateWebHook(data *GitWebHookArguments) error {
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
func (g *GitFakeProvider) IsGitHub() bool {
	return false
}

// IsGitHub returns true if gitea
func (g *GitFakeProvider) IsGitea() bool {
	return false
}

// IsBitbucketCloud returns true if bitbucket cloud
func (g *GitFakeProvider) IsBitbucketCloud() bool {
	return false
}

// IsBitbucketServer returns true if bitbucket server
func (g *GitFakeProvider) IsBitbucketServer() bool {
	return false
}

// IsGerrit returns true if gerrit
func (g *GitFakeProvider) IsGerrit() bool {
	return false
}

// Kind returns the kind
func (g *GitFakeProvider) Kind() string {
	return KindGitFake
}

// GetIssue get an issue
func (g *GitFakeProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	panic("implement me")
}

// IssueURL get an issue URL
func (g *GitFakeProvider) IssueURL(org string, name string, number int, isPull bool) string {
	panic("implement me")
}

// SearchIssues search issues
func (g *GitFakeProvider) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {
	panic("implement me")
}

// SearchIssuesClosedSince search issues closed since
func (g *GitFakeProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	panic("implement me")
}

// CreateIssue create an issue
func (g *GitFakeProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	panic("implement me")
}

// HasIssues returns true if has issues
func (g *GitFakeProvider) HasIssues() bool {
	panic("implement me")
}

// AddPRComment add a comment to a PR
func (g *GitFakeProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	panic("implement me")
}

// CreateIssueComment create a comment on an issue
func (g *GitFakeProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	panic("implement me")
}

// UpdateRelease update a release
func (g *GitFakeProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	panic("implement me")
}

// ListReleases list the releases
func (g *GitFakeProvider) ListReleases(org string, name string) ([]*GitRelease, error) {
	panic("implement me")
}

// GetContent gets the content for a file
func (g *GitFakeProvider) GetContent(org string, name string, path string, ref string) (*GitFileContent, error) {
	panic("implement me")
}

// JenkinsWebHookPath returns the path for jenkins webhooks
func (g *GitFakeProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/fake-webhook/"
}

// Label return the label
func (g *GitFakeProvider) Label() string {
	return "fake"
}

// ServerURL returns the server URL
func (g *GitFakeProvider) ServerURL() string {
	return g.serverURL
}

// BranchArchiveURL returns the branch archive URL
func (g *GitFakeProvider) BranchArchiveURL(org string, name string, branch string) string {
	panic("implement me")
}

// CurrentUsername returns the current user name
func (g *GitFakeProvider) CurrentUsername() string {
	return g.GitUser.Login
}

// UserAuth returns the current user auth
func (g *GitFakeProvider) UserAuth() auth.UserAuth {
	return g.userAuth
}

// UserInfo returns the user info for the given user name
func (g *GitFakeProvider) UserInfo(username string) *GitUser {
	panic("implement me")
}

// AddCollaborator adds a collaborator
func (g *GitFakeProvider) AddCollaborator(string, string, string) error {
	panic("implement me")
}

// ListInvitations list invitations
func (g *GitFakeProvider) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	panic("implement me")
}

// AcceptInvitation accepts invitation
func (g *GitFakeProvider) AcceptInvitation(int64) (*github.Response, error) {
	panic("implement me")
}

func (g *GitFakeProvider) notFound() error {
	return fmt.Errorf("Not found")
}
