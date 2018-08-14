package gits

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
)

const jenkinsWebhookPath = "/jenkins-webhook/"

type FakeProviderType int

const (
	GitHub FakeProviderType = iota
	Gitlab
	Gitea
	BitbucketCloud
	BitbucketServer
	Gerrit
)

type CommitStatus string

const (
	CommitStatusPending CommitStatus = "pending"
	CommitSatusSuccess               = "success"
	CommitStatusError                = "error"
	CommitStatusFailure              = "failure"
)

type FakeCommit struct {
	Commit *GitCommit
	Status CommitStatus
}

type FakePullRequest struct {
	PullRequest *GitPullRequest
	Commits     []*FakeCommit
	Comment     string
}

type FakeIssue struct {
	Issue   *GitIssue
	Comment string
}

type FakeRepository struct {
	GitRepo      *GitRepository
	PullRequests map[int]*FakePullRequest
	Issues       map[int]*FakeIssue
	Commits      []*FakeCommit
	issueCount   int
	Releases     map[string]*GitRelease
}

type FakeProvider struct {
	Server             auth.AuthServer
	User               auth.UserAuth
	Organizations      []GitOrganisation
	Repositories       map[string][]*FakeRepository
	ForkedRepositories map[string][]*FakeRepository
	Type               FakeProviderType
	Users              []*GitUser
}

func (f *FakeProvider) ListOrganisations() ([]GitOrganisation, error) {
	return f.Organizations, nil
}

func (f *FakeProvider) ListRepositories(org string) ([]*GitRepository, error) {
	repos, ok := f.Repositories[org]
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", org)
	}
	gitRepos := []*GitRepository{}
	for _, repo := range repos {
		gitRepos = append(gitRepos, repo.GitRepo)
	}
	return gitRepos, nil
}

func (f *FakeProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	gitRepo := &GitRepository{
		Name: name,
	}

	repo := &FakeRepository{
		GitRepo:      gitRepo,
		PullRequests: nil,
	}
	f.Repositories[org] = append(f.Repositories[org], repo)
	return gitRepo, nil
}

func (f *FakeProvider) GetRepository(org string, name string) (*GitRepository, error) {
	repos, ok := f.Repositories[org]
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", org)
	}
	for _, repo := range repos {
		if repo.GitRepo.Name == name {
			return repo.GitRepo, nil
		}
	}
	return nil, fmt.Errorf("repository '%s' not found within the organization '%s'", name, org)
}

func (f *FakeProvider) DeleteRepository(org string, name string) error {
	for i, repo := range f.Repositories[org] {
		if repo.GitRepo.Name == name {
			f.Repositories[org] = append(f.Repositories[org][:i], f.Repositories[org][i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("repository '%s' not found within the organization '%s'", name, org)
}

func (f *FakeProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	for _, repo := range f.Repositories[originalOrg] {
		if repo.GitRepo.Name == name {
			f.ForkedRepositories[destinationOrg] = append(f.ForkedRepositories[destinationOrg], repo)
			return repo.GitRepo, nil
		}
	}
	return nil, fmt.Errorf("repository '%s' not found within the organization '%s'", name, originalOrg)
}

func (f *FakeProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	for _, repo := range f.Repositories[org] {
		if repo.GitRepo.Name == name {
			repo.GitRepo.Name = newName
			return repo.GitRepo, nil
		}
	}
	return nil, fmt.Errorf("repository '%s' not found within the organization '%s'", name, org)
}

func (f *FakeProvider) ValidateRepositoryName(org string, name string) error {
	for _, repo := range f.Repositories[org] {
		if repo.GitRepo.Name == name {
			return nil
		}
	}
	return fmt.Errorf("repository '%s' not found within the organization '%s'", name, org)
}

func (f *FakeProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	org := data.GitRepositoryInfo.Organisation
	repoName := data.GitRepositoryInfo.Name
	repos, ok := f.Repositories[org]
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", org)
	}

	var repo *FakeRepository
	for _, r := range repos {
		if r.GitRepo.Name == repoName {
			repo = r
		}
	}
	if repo == nil {
		return nil, fmt.Errorf("repository '%s' not found", repoName)
	}

	repo.issueCount += 1
	number := repo.issueCount
	pr := &GitPullRequest{
		URL: "",
		Author: &GitUser{
			URL:       "",
			Login:     "",
			Name:      "",
			Email:     "",
			AvatarURL: "",
		},
		Owner:          org,
		Repo:           repoName,
		Number:         &number,
		Mergeable:      nil,
		Merged:         nil,
		HeadRef:        nil,
		State:          nil,
		StatusesURL:    nil,
		IssueURL:       nil,
		DiffURL:        nil,
		MergeCommitSHA: nil,
		ClosedAt:       nil,
		MergedAt:       nil,
		LastCommitSha:  "",
		Title:          data.Title,
		Body:           data.Body,
	}

	repo.PullRequests[number] = &FakePullRequest{PullRequest: pr}
	return pr, nil
}

func (f *FakeProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
	return nil
}

func (f *FakeProvider) GetPullRequest(owner string, repo *GitRepositoryInfo, number int) (*GitPullRequest, error) {
	repos, ok := f.Repositories[owner]
	if !ok {
		return nil, fmt.Errorf("no repositories found for '%s'", owner)
	}
	repoName := repo.Name
	for _, r := range repos {
		if r.GitRepo.Name == repoName {
			pr, ok := r.PullRequests[number]
			if !ok {
				return nil, fmt.Errorf("pull request with id '%d' not found", number)
			}
			return pr.PullRequest, nil
		}
	}

	return nil, fmt.Errorf("repository with name '%s' not found", repoName)
}

func (f *FakeProvider) GetPullRequestCommits(owner string, repo *GitRepositoryInfo, number int) ([]*GitCommit, error) {
	repos, ok := f.Repositories[owner]
	if !ok {
		return nil, fmt.Errorf("no repositories found for '%s'", owner)
	}
	repoName := repo.Name
	for _, r := range repos {
		if r.GitRepo.Name == repoName {
			pr, ok := r.PullRequests[number]
			if !ok {
				return nil, fmt.Errorf("pull request with id '%d' not found", number)
			}
			commits := []*GitCommit{}
			for _, commit := range pr.Commits {
				commits = append(commits, commit.Commit)
			}
			return commits, nil
		}
	}
	return nil, fmt.Errorf("repository with name '%s' not found", repoName)
}

func (f *FakeProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	owner := pr.Owner
	repos, ok := f.Repositories[owner]
	if !ok {
		return "", fmt.Errorf("no repositories found for '%s'", owner)
	}
	repoName := pr.Repo
	number := *pr.Number
	for _, r := range repos {
		if r.GitRepo.Name == repoName {
			pr, ok := r.PullRequests[number]
			if !ok {
				return "", fmt.Errorf("pull request with id '%d' not found", number)
			}
			len := len(pr.Commits)
			if len < 1 {
				return "", errors.New("pull request has no commits")
			}
			lastCommit := pr.Commits[len-1]
			return string(lastCommit.Status), nil
		}
	}
	return "", fmt.Errorf("repository with name '%s' not found", repoName)
}

func (f *FakeProvider) ListCommitStatus(org string, repoName string, sha string) ([]*GitRepoStatus, error) {
	repos, ok := f.Repositories[org]
	if !ok {
		return nil, fmt.Errorf("no repositories found for '%s'", org)
	}
	var repo *FakeRepository
	for _, r := range repos {
		if r.GitRepo.Name == repoName {
			repo = r
		}
	}

	if repo == nil {
		return nil, fmt.Errorf("repository with name '%s' not found", repoName)
	}

	answer := []*GitRepoStatus{}
	for _, commit := range repo.Commits {
		if commit.Commit.SHA == sha {
			status := &GitRepoStatus{
				ID:          commit.Commit.SHA,
				URL:         commit.Commit.URL,
				State:       string(commit.Status),
				Description: commit.Commit.Message,
			}
			answer = append(answer, status)
		}
	}
	return answer, nil
}

func (f *FakeProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	owner := pr.Owner
	repos, ok := f.Repositories[owner]
	if !ok {
		return fmt.Errorf("no repositories found for '%s'", owner)
	}
	repoName := pr.Repo
	number := *pr.Number
	for _, r := range repos {
		if r.GitRepo.Name == repoName {
			_, ok := r.PullRequests[number]
			if !ok {
				return fmt.Errorf("pull request with id '%d' not found", number)
			}
			delete(r.PullRequests, number)
			return nil
		}
	}
	return fmt.Errorf("repository with name '%s' not found", repoName)
}

func (f *FakeProvider) CreateWebHook(data *GitWebHookArguments) error {
	return nil
}

func (f *FakeProvider) IsGitHub() bool {
	return f.Type == GitHub
}

func (f *FakeProvider) IsGitea() bool {
	return f.Type == Gitea
}

func (f *FakeProvider) IsBitbucketCloud() bool {
	return f.Type == BitbucketCloud
}

func (f *FakeProvider) IsBitbucketServer() bool {
	return f.Type == BitbucketServer
}

func (f *FakeProvider) IsGerrit() bool {
	return f.Type == Gerrit
}

func (f *FakeProvider) Kind() string {
	switch f.Type {
	case GitHub:
		return KindGitHub
	case Gitlab:
		return KindGitlab
	case Gitea:
		return KindGitea
	case BitbucketCloud:
		return KindBitBucketCloud
	case BitbucketServer:
		return KindBitBucketServer
	default:
		return KindUnknown
	}
}

func (f *FakeProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	repos, ok := f.Repositories[org]
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", org)
	}
	for _, repo := range repos {
		if repo.GitRepo.Name == name {
			issue, ok := repo.Issues[number]
			if !ok {
				return nil, fmt.Errorf("no issue found with ID '%d'", number)
			}
			return issue.Issue, nil
		}
	}
	return nil, fmt.Errorf("no issue found with name '%s'", name)
}

func (f *FakeProvider) IssueURL(org string, name string, number int, isPull bool) string {
	serverPrefix := f.Server.URL
	if !strings.HasPrefix(serverPrefix, "https://") {
		serverPrefix = "https://" + serverPrefix
	}
	path := "issues"
	if isPull {
		path = "pull"
	}
	url := util.UrlJoin(serverPrefix, org, name, path, strconv.Itoa(number))
	return url
}

func (f *FakeProvider) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {
	repos, ok := f.Repositories[org]
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", org)
	}
	for _, repo := range repos {
		if repo.GitRepo.Name == name {
			answer := []*GitIssue{}
			for _, issue := range repo.Issues {
				answer = append(answer, issue.Issue)
			}
			return answer, nil
		}
	}
	return nil, fmt.Errorf("repository with name '%s' not found", name)
}

func (f *FakeProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	issues, err := f.SearchIssues(org, name, "")
	if err != nil {
		return nil, err
	}

	answer := []*GitIssue{}
	for _, issue := range issues {
		closedTime := issue.ClosedAt
		if closedTime.After(t) {
			answer = append(answer, issue)
		}
	}
	return answer, nil
}

func (f *FakeProvider) CreateIssue(owner string, repoName string, issue *GitIssue) (*GitIssue, error) {
	repos, ok := f.Repositories[owner]
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", owner)
	}
	for _, repo := range repos {
		if repo.GitRepo.Name == repoName {
			repo.issueCount += 1
			number := repo.issueCount
			issue.Number = &number
			newIssue := &FakeIssue{
				Issue:   issue,
				Comment: "",
			}
			repo.Issues[number] = newIssue
			return issue, nil
		}
	}
	return nil, fmt.Errorf("repository with name '%s' not found", repoName)
}

func (f *FakeProvider) HasIssues() bool {
	return true
}

func (f *FakeProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	owner := pr.Owner
	repos, ok := f.Repositories[owner]
	if !ok {
		return fmt.Errorf("no repositories found for '%s'", owner)
	}
	repoName := pr.Repo
	number := *pr.Number
	for _, r := range repos {
		if r.GitRepo.Name == repoName {
			pr, ok := r.PullRequests[number]
			if !ok {
				return fmt.Errorf("pull request with id '%d' not found", number)
			}
			pr.Comment = comment
			return nil
		}
	}
	return fmt.Errorf("repository with name '%s' not found", repoName)
}

func (f *FakeProvider) CreateIssueComment(owner string, repoName string, number int, comment string) error {
	repos, ok := f.Repositories[owner]
	if !ok {
		return fmt.Errorf("no repositories found for '%s'", owner)
	}
	for _, r := range repos {
		if r.GitRepo.Name == repoName {
			issue, ok := r.Issues[number]
			if !ok {
				return fmt.Errorf("issue with id '%d' not found", number)
			}
			issue.Comment = comment
			return nil
		}
	}
	return fmt.Errorf("repository with name '%s' not found", repoName)
}

func (f *FakeProvider) UpdateRelease(owner string, repoName string, tag string, releaseInfo *GitRelease) error {
	repos, ok := f.Repositories[owner]
	if !ok {
		return fmt.Errorf("organization '%s' not found", owner)
	}

	for _, repo := range repos {
		if repo.GitRepo.Name == repoName {
			repo.Releases[tag] = releaseInfo
			return nil
		}
	}
	return fmt.Errorf("repository with name '%s' not found", repoName)
}

func (f *FakeProvider) ListReleases(org string, name string) ([]*GitRelease, error) {
	repos, ok := f.Repositories[org]
	if !ok {
		return nil, fmt.Errorf("organization '%s' not found", org)
	}

	for _, repo := range repos {
		if repo.GitRepo.Name == name {
			releases := []*GitRelease{}
			for _, release := range repo.Releases {
				releases = append(releases, release)
			}
			return releases, nil
		}
	}
	return nil, fmt.Errorf("repository with name '%s' not found", name)
}

func (f *FakeProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return jenkinsWebhookPath
}

func (f *FakeProvider) Label() string {
	return f.Server.Label()
}

func (f *FakeProvider) ServerURL() string {
	return f.Server.URL
}

func (f *FakeProvider) CurrentUsername() string {
	return f.User.Username
}

func (f *FakeProvider) UserAuth() auth.UserAuth {
	return f.User
}

func (f *FakeProvider) UserInfo(username string) *GitUser {
	for _, user := range f.Users {
		if user.Name == username {
			return user
		}
	}
	return nil
}
