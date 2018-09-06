package gits

import (
	"context"
	"time"

	gerrit "github.com/andygrunwald/go-gerrit"
	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/log"
)

type GerritProvider struct {
	Client   *gerrit.Client
	Username string
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
	Git    Gitter
}

func NewGerritProvider(server *auth.AuthServer, user *auth.UserAuth, git Gitter) (GitProvider, error) {
	return nil, nil
}

func (p *GerritProvider) ListRepositories(org string) ([]*GitRepository, error) {
	return nil, nil
}

func (p *GerritProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	return nil, nil
}

func (p *GerritProvider) GetRepository(org string, name string) (*GitRepository, error) {
	return nil, nil
}

func (p *GerritProvider) DeleteRepository(org string, name string) error {
	return nil
}

func (p *GerritProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	return nil, nil
}

func (p *GerritProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	return nil, nil
}

func (p *GerritProvider) ValidateRepositoryName(org string, name string) error {
	return nil
}

func (p *GerritProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	return nil, nil
}

func (p *GerritProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
	return nil
}

func (p *GerritProvider) GetPullRequest(owner string, repo *GitRepositoryInfo, number int) (*GitPullRequest, error) {
	return nil, nil
}

func (p *GerritProvider) GetPullRequestCommits(owner string, repo *GitRepositoryInfo, number int) ([]*GitCommit, error) {
	return nil, nil
}

func (p *GerritProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	return "", nil
}

func (p *GerritProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	return nil, nil
}

func (p *GerritProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	return nil
}

func (p *GerritProvider) CreateWebHook(data *GitWebHookArguments) error {
	return nil
}

func (p *GerritProvider) IsGitHub() bool {
	return false
}

func (p *GerritProvider) IsGitea() bool {
	return false
}

func (p *GerritProvider) IsBitbucketCloud() bool {
	return false
}

func (p *GerritProvider) IsBitbucketServer() bool {
	return false
}

func (p *GerritProvider) IsGerrit() bool {
	return true
}

func (p *GerritProvider) Kind() string {
	return "gerrit"
}

func (p *GerritProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	return nil, nil
}

func (p *GerritProvider) IssueURL(org string, name string, number int, isPull bool) string {
	return ""
}

func (p *GerritProvider) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {
	return nil, nil
}

func (p *GerritProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	return nil, nil
}

func (p *GerritProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	return nil, nil
}

func (p *GerritProvider) HasIssues() bool {
	return false
}

func (p *GerritProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	return nil
}

func (p *GerritProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	return nil
}

func (p *GerritProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	return nil
}

func (p *GerritProvider) ListReleases(org string, name string) ([]*GitRelease, error) {
	return nil, nil
}

func (p *GerritProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return ""
}

func (p *GerritProvider) Label() string {
	return ""
}

func (p *GerritProvider) ServerURL() string {
	return ""
}

func (p *GerritProvider) BranchArchiveURL(org string, name string, branch string) string {
	return ""
}

func (p *GerritProvider) CurrentUsername() string {
	return ""
}

func (p *GerritProvider) UserAuth() auth.UserAuth {
	return auth.UserAuth{}
}

func (p *GerritProvider) UserInfo(username string) *GitUser {
	return nil
}

func (p *GerritProvider) AddCollaborator(user string, repo string) error {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for gerrit. Please add user: %v as a collaborator to this project.\n", user)
	return nil
}

func (p *GerritProvider) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for gerrit.\n")
	return []*github.RepositoryInvitation{}, &github.Response{}, nil
}

func (p *GerritProvider) AcceptInvitation(ID int64) (*github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for gerrit.\n")
	return &github.Response{}, nil
}
