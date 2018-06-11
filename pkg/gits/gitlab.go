package gits

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/wbrefvem/go-gitlab"
)

type GitlabProvider struct {
	Username string
	Client   *gitlab.Client
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
}

func NewGitlabProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	c := gitlab.NewClient(nil, user.ApiToken)
	return withGitlabClient(server, user, c)
}

// Used by unit tests to inject a mocked client
func withGitlabClient(server *auth.AuthServer, user *auth.UserAuth, client *gitlab.Client) (GitProvider, error) {
	provider := &GitlabProvider{
		Server:   *server,
		User:     *user,
		Username: user.Username,
		Client:   client,
	}
	return provider, nil
}

func (g *GitlabProvider) ListRepositories(org string) ([]*GitRepository, error) {
	result, _, err := getRepositories(g.Client, g.Username, org)
	if err != nil {
		return nil, err
	}

	var repos []*GitRepository
	for _, p := range result {
		repos = append(repos, fromGitlabProject(p))
	}
	return repos, nil
}

func (g *GitlabProvider) ListReleases(org string, name string) ([]*GitRelease, error) {
	answer := []*GitRelease{}
	// TODO
	return answer, nil
}

func getRepositories(g *gitlab.Client, username string, org string) ([]*gitlab.Project, *gitlab.Response, error) {
	if org == "" {
		return g.Projects.ListUserProjects(username, &gitlab.ListProjectsOptions{Owned: gitlab.Bool(true)})
	}
	return g.Groups.ListGroupProjects(org, nil)
}

func fromGitlabProject(p *gitlab.Project) *GitRepository {
	return &GitRepository{
		Name:     p.Name,
		HTMLURL:  p.WebURL,
		SSHURL:   p.SSHURLToRepo,
		CloneURL: p.SSHURLToRepo,
		Fork:     p.ForkedFromProject != nil,
	}
}

func (g *GitlabProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	visibility := gitlab.PublicVisibility
	if private {
		visibility = gitlab.PrivateVisibility
	}

	p := &gitlab.CreateProjectOptions{
		Name:       &name,
		Visibility: &visibility,
	}

	project, _, err := g.Client.Projects.CreateProject(p)
	if err != nil {
		return nil, err
	}
	return fromGitlabProject(project), nil
}

func projectId(org, username, repoName string) string {
	return fmt.Sprintf("%s/%s", owner(org, username), repoName)
}

func owner(org, username string) string {
	if org == "" {
		return username
	}
	return org
}

func (g *GitlabProvider) GetRepository(org, name string) (*GitRepository, error) {
	pid := projectId(org, g.Username, name)
	project, response, err := g.Client.Projects.GetProject(pid)
	if err != nil {
		return nil, fmt.Errorf("%v", response.Request.URL)
	}
	return fromGitlabProject(project), nil
}

func (g *GitlabProvider) ListOrganisations() ([]GitOrganisation, error) {
	groups, _, err := g.Client.Groups.ListGroups(nil)
	if err != nil {
		return nil, err
	}

	var organizations []GitOrganisation
	for _, v := range groups {
		organizations = append(organizations, GitOrganisation{v.Path})
	}
	return organizations, nil
}

func (g *GitlabProvider) DeleteRepository(org, name string) error {
	pid := projectId(org, g.Username, name)

	_, err := g.Client.Projects.DeleteProject(pid)
	if err != nil {
		return fmt.Errorf("failed to delete repository %s due to: %s", pid, err)
	}
	return err
}

func (g *GitlabProvider) ForkRepository(originalOrg, name, destinationOrg string) (*GitRepository, error) {
	pid := projectId(originalOrg, g.Username, name)
	project, _, err := g.Client.Projects.ForkProject(pid)
	if err != nil {
		return nil, err
	}

	return fromGitlabProject(project), nil
}

func (g *GitlabProvider) RenameRepository(org, name, newName string) (*GitRepository, error) {
	pid := projectId(org, g.Username, name)
	options := &gitlab.EditProjectOptions{
		Name: &newName,
	}

	project, _, err := g.Client.Projects.EditProject(pid, options)
	if err != nil {
		return nil, err
	}
	return fromGitlabProject(project), nil
}

func (g *GitlabProvider) ValidateRepositoryName(org, name string) error {
	pid := projectId(org, g.Username, name)
	_, r, err := g.Client.Projects.GetProject(pid)

	if err == nil {
		return fmt.Errorf("repository %s already exists", pid)
	}
	if r != nil && r.StatusCode == 404 {
		return nil
	}
	return err
}

func (g *GitlabProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	owner := data.GitRepositoryInfo.Organisation
	repo := data.GitRepositoryInfo.Name
	title := data.Title
	body := data.Body
	head := data.Head
	base := data.Base

	o := &gitlab.CreateMergeRequestOptions{
		Title:        &title,
		Description:  &body,
		SourceBranch: &head,
		TargetBranch: &base,
	}

	pid := projectId(owner, g.Username, repo)
	mr, _, err := g.Client.MergeRequests.CreateMergeRequest(pid, o)
	if err != nil {
		return nil, err
	}

	return fromMergeRequest(mr, owner, repo), nil
}

func fromMergeRequest(mr *gitlab.MergeRequest, owner, repo string) *GitPullRequest {
	return &GitPullRequest{
		Author: &GitUser{
			Login: mr.Author.Username,
		},
		URL:    mr.WebURL,
		Owner:  owner,
		Repo:   repo,
		Number: &mr.IID,
		State:  &mr.State,
		Title:  mr.Title,
		Body:   mr.Description,
	}
}

func (g *GitlabProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
	owner := pr.Owner
	repo := pr.Repo

	pid := projectId(owner, g.Username, repo)
	mr, _, err := g.Client.MergeRequests.GetMergeRequest(pid, *pr.Number)
	if err != nil {
		return err
	}

	*pr = *fromMergeRequest(mr, owner, repo)
	return nil
}

func (p *GitlabProvider) GetPullRequest(owner string, repo *GitRepositoryInfo, number int) (*GitPullRequest, error) {
	pr := &GitPullRequest{
		Owner:  owner,
		Repo:   repo.Name,
		Number: &number,
	}
	err := p.UpdatePullRequestStatus(pr)

	existing := p.UserInfo(pr.Author.Login)
	if existing != nil && existing.Email != "" {
		pr.Author = existing
	}

	return pr, err
}

func (p *GitlabProvider) GetPullRequestCommits(owner string, repository *GitRepositoryInfo, number int) ([]*GitCommit, error) {
	repo := repository.Name
	pid := projectId(owner, p.Username, repo)
	commits, _, err := p.Client.MergeRequests.GetMergeRequestCommits(pid, number, nil)

	if err != nil {
		return nil, err
	}

	answer := []*GitCommit{}

	for _, commit := range commits {
		if commit == nil {
			continue
		}
		summary := &GitCommit{
			Message: commit.Message,
			SHA:     commit.ID,
			Author: &GitUser{
				Email: commit.AuthorEmail,
			},
		}
		answer = append(answer, summary)
	}

	return answer, nil
}

func (g *GitlabProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	owner := pr.Owner
	repo := pr.Repo

	ref := pr.LastCommitSha
	if ref == "" {
		return "", fmt.Errorf("missing String for LastCommitSha %#v", pr)
	}

	pid := projectId(owner, g.Username, repo)
	c, _, err := g.Client.Commits.GetCommitStatuses(pid, ref, nil)
	if err != nil {
		return "", err
	}

	for _, result := range c {
		if result.Status != "" {
			return result.Status, nil
		}
	}
	return "", fmt.Errorf("could not find a status for repository %s with ref %s", pid, ref)
}

func (g *GitlabProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	pid := projectId(org, g.Username, repo)
	c, _, err := g.Client.Commits.GetCommitStatuses(pid, sha, nil)
	if err != nil {
		return nil, err
	}

	var statuses []*GitRepoStatus

	for _, result := range c {
		statuses = append(statuses, fromCommitStatus(result))
	}

	return statuses, nil
}

func fromCommitStatus(status *gitlab.CommitStatus) *GitRepoStatus {
	return &GitRepoStatus{
		ID:          string(status.ID),
		URL:         status.TargetURL,
		State:       status.Status,
		Description: status.Description,
	}
}

func (g *GitlabProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	pid := projectId(pr.Owner, g.Username, pr.Repo)

	opt := &gitlab.AcceptMergeRequestOptions{MergeCommitMessage: &message}

	_, _, err := g.Client.MergeRequests.AcceptMergeRequest(pid, *pr.Number, opt)
	return err
}

func (g *GitlabProvider) CreateWebHook(data *GitWebHookArguments) error {
	pid := projectId(data.Owner, g.Username, data.Repo.Name)

	opt := &gitlab.AddProjectHookOptions{
		URL:   &data.URL,
		Token: &data.Secret,
	}

	_, _, err := g.Client.Projects.AddProjectHook(pid, opt)
	return err
}

func (g *GitlabProvider) SearchIssues(org, repo, query string) ([]*GitIssue, error) {
	opt := &gitlab.ListProjectIssuesOptions{Search: &query}
	return g.searchIssuesWithOptions(org, repo, opt)
}

func (g *GitlabProvider) SearchIssuesClosedSince(org string, repo string, t time.Time) ([]*GitIssue, error) {
	closed := "closed"
	opt := &gitlab.ListProjectIssuesOptions{State: &closed}
	issues, err := g.searchIssuesWithOptions(org, repo, opt)
	if err != nil {
		return issues, err
	}
	return FilterIssuesClosedSince(issues, t), nil
}

func (g *GitlabProvider) searchIssuesWithOptions(org string, repo string, opt *gitlab.ListProjectIssuesOptions) ([]*GitIssue, error) {
	pid := projectId(org, g.Username, repo)
	issues, _, err := g.Client.Issues.ListProjectIssues(pid, opt)
	if err != nil {
		return nil, err
	}
	return fromGitlabIssues(issues, owner(org, g.Username), repo), nil
}

func (g *GitlabProvider) GetIssue(org, repo string, number int) (*GitIssue, error) {
	owner := owner(org, g.Username)
	pid := projectId(org, g.Username, repo)

	issue, _, err := g.Client.Issues.GetIssue(pid, number)
	if err != nil {
		return nil, err
	}
	return fromGitlabIssue(issue, owner, repo), nil
}

func (g *GitlabProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	labels := []string{}
	for _, label := range issue.Labels {
		name := label.Name
		if name != "" {
			labels = append(labels, name)
		}
	}

	opt := &gitlab.CreateIssueOptions{
		Title:       &issue.Title,
		Description: &issue.Body,
		Labels:      labels,
	}

	pid := projectId(owner, g.Username, repo)
	gitlabIssue, _, err := g.Client.Issues.CreateIssue(pid, opt)
	if err != nil {
		return nil, err
	}

	return fromGitlabIssue(gitlabIssue, owner, repo), nil
}

func fromGitlabIssues(issues []*gitlab.Issue, owner, repo string) []*GitIssue {
	var result []*GitIssue

	for _, v := range issues {
		result = append(result, fromGitlabIssue(v, owner, repo))
	}
	return result
}

func fromGitlabIssue(issue *gitlab.Issue, owner, repo string) *GitIssue {
	var labels []GitLabel
	for _, v := range issue.Labels {
		labels = append(labels, GitLabel{Name: v})
	}

	return &GitIssue{
		Number:    &issue.IID,
		URL:       issue.WebURL,
		Owner:     owner,
		Repo:      repo,
		Title:     issue.Title,
		Body:      issue.Description,
		Labels:    labels,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		ClosedAt:  issue.ClosedAt,
	}
}

func (g *GitlabProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	owner := pr.Owner
	repo := pr.Repo

	opt := &gitlab.CreateMergeRequestNoteOptions{Body: &comment}

	pid := projectId(owner, g.Username, repo)
	_, _, err := g.Client.Notes.CreateMergeRequestNote(pid, *pr.Number, opt)
	return err
}

func (g *GitlabProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	opt := &gitlab.CreateIssueNoteOptions{Body: &comment}

	pid := projectId(owner, g.Username, repo)
	_, _, err := g.Client.Notes.CreateIssueNote(pid, number, opt)
	return err
}

func (g *GitlabProvider) HasIssues() bool {
	return true
}

func (g *GitlabProvider) IsGitHub() bool {
	return false
}

func (g *GitlabProvider) IsGitea() bool {
	return false
}

func (g *GitlabProvider) IsBitbucketCloud() bool {
	return false
}

func (g *GitlabProvider) IsBitbucketServer() bool {
	return false
}

func (g *GitlabProvider) Kind() string {
	return "gitlab"
}

func (g *GitlabProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/gitlab/notify_commit"
}

func (g *GitlabProvider) Label() string {
	return g.Server.Label()
}

func (p *GitlabProvider) ServerURL() string {
	return p.Server.URL
}

func (p *GitlabProvider) CurrentUsername() string {
	return p.Username
}

func (p *GitlabProvider) UserAuth() auth.UserAuth {
	return p.User
}

func (p *GitlabProvider) UserInfo(username string) *GitUser {
	users, _, err := p.Client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &username})

	if err != nil || len(users) == 0 {
		return nil
	}

	user := users[0]

	return &GitUser{
		Login:     username,
		URL:       user.WebsiteURL,
		AvatarURL: user.AvatarURL,
		Name:      user.Name,
		Email:     user.Email,
	}
}

func (g *GitlabProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	return nil
}

func (p *GitlabProvider) IssueURL(org string, name string, number int, isPull bool) string {
	return ""
}

// GitlabAccessTokenURL returns the URL to click on to generate a personal access token for the git provider
func GitlabAccessTokenURL(url string) string {
	return util.UrlJoin(url, "/profile/personal_access_tokens")
}
