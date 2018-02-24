package gits

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/auth"
	"golang.org/x/oauth2"
)

type GitHubProvider struct {
	Username string
	Client   *github.Client
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
}

func NewGitHubProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	ctx := context.Background()

	provider := GitHubProvider{
		Server:   *server,
		User:     *user,
		Context:  ctx,
		Username: user.Username,
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: user.ApiToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	provider.Client = github.NewClient(tc)
	return &provider, nil
}

func (p *GitHubProvider) ListOrganisations() ([]GitOrganisation, error) {
	answer := []GitOrganisation{}
	pageSize := 100
	options := github.ListOptions{
		Page:    0,
		PerPage: pageSize,
	}
	for {
		orgs, _, err := p.Client.Organizations.List(p.Context, "", &options)
		if err != nil {
			return answer, err
		}

		for _, org := range orgs {
			name := org.Login
			if name != nil {
				o := GitOrganisation{
					Login: *name,
				}
				answer = append(answer, o)
			}
		}
		if len(orgs) < pageSize || len(orgs) == 0 {
			break
		}
		options.Page += 1
	}
	return answer, nil
}

func (p *GitHubProvider) ListRepositories(org string) ([]*GitRepository, error) {
	owner := org
	if owner == "" {
		owner = p.Username
	}
	answer := []*GitRepository{}
	pageSize := 100
	options := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: pageSize,
		},
	}
	for {
		repos, _, err := p.Client.Repositories.List(p.Context, owner, options)
		if err != nil {
			return answer, err
		}
		for _, repo := range repos {
			answer = append(answer, toGitHubRepo(asText(repo.Name), repo))
		}
		if len(repos) < pageSize || len(repos) == 0 {
			break
		}
		options.ListOptions.Page += 1
	}
	return answer, nil
}

func (p *GitHubProvider) GetRepository(org string, name string) (*GitRepository, error) {
	repo, _, err := p.Client.Repositories.Get(p.Context, org, name)
	if err != nil {
		return nil, fmt.Errorf("Failed to get repository %s/%s due to: %s", org, name, err)
	}
	return toGitHubRepo(name, repo), nil
}

func (p *GitHubProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	repoConfig := &github.Repository{
		Name:    github.String(name),
		Private: github.Bool(private),
	}
	repo, _, err := p.Client.Repositories.Create(p.Context, org, repoConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create repository %s/%s due to: %s", org, name, err)
	}
	return toGitHubRepo(name, repo), nil
}

func (p *GitHubProvider) DeleteRepository(org string, name string) error {
	owner := org
	if owner == "" {
		owner = p.Username
	}
	_, err := p.Client.Repositories.Delete(p.Context, owner, name)
	if err != nil {
		return fmt.Errorf("Failed to delete repository %s/%s due to: %s", owner, name, err)
	}
	return err
}

func toGitHubRepo(name string, repo *github.Repository) *GitRepository {
	return &GitRepository{
		Name:             name,
		AllowMergeCommit: asBool(repo.AllowMergeCommit),
		CloneURL:         asText(repo.CloneURL),
		HTMLURL:          asText(repo.HTMLURL),
		SSHURL:           asText(repo.SSHURL),
	}
}

func (p *GitHubProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	repoConfig := &github.RepositoryCreateForkOptions{}
	if destinationOrg != "" {
		repoConfig.Organization = destinationOrg
	}
	repo, _, err := p.Client.Repositories.CreateFork(p.Context, originalOrg, name, repoConfig)
	if err != nil {
		msg := ""
		if destinationOrg != "" {
			msg = fmt.Sprintf(" to %s", destinationOrg)
		}
		owner := destinationOrg
		if owner == "" {
			owner = p.Username
		}
		if strings.Contains(err.Error(), "try again later") {
			fmt.Printf("Waiting for the fork of %s/%s to appear...\n", owner, name)
			// lets wait for the fork to occur...
			start := time.Now()
			deadline := start.Add(time.Minute)
			for {
				time.Sleep(5 * time.Second)
				repo, _, err = p.Client.Repositories.Get(p.Context, owner, name)
				if repo != nil && err == nil {
					break
				}
				t := time.Now()
				if t.After(deadline) {
					return nil, fmt.Errorf("Gave up waiting for Repository %s/%s to appear: %s", owner, name, err)
				}
			}
		} else {
			return nil, fmt.Errorf("Failed to fork repository %s/%s%s due to: %s", originalOrg, name, msg, err)
		}
	}
	answer := &GitRepository{
		Name:             name,
		AllowMergeCommit: asBool(repo.AllowMergeCommit),
		CloneURL:         asText(repo.CloneURL),
		HTMLURL:          asText(repo.HTMLURL),
		SSHURL:           asText(repo.SSHURL),
	}
	return answer, nil
}

func (p *GitHubProvider) CreateWebHook(data *GitWebHookArguments) error {
	owner := data.Owner
	if owner == "" {
		owner = p.Username
	}
	repo := data.Repo
	if repo == "" {
		return fmt.Errorf("Missing property Repo")
	}
	webhookUrl := data.URL
	if repo == "" {
		return fmt.Errorf("Missing property URL")
	}
	hooks, _, err := p.Client.Repositories.ListHooks(p.Context, owner, repo, nil)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		c := hook.Config["url"]
		s, ok := c.(string)
		if ok && s == webhookUrl {
			fmt.Printf("Already has a webhook registered for %s\n", webhookUrl)
			return nil
		}
	}
	config := map[string]interface{}{
		"url":          webhookUrl,
		"content_type": "json",
	}
	if data.Secret != "" {
		config["secret"] = data.Secret
	}
	hook := &github.Hook{
		Name:   github.String("web"),
		Config: config,
		Events: []string{"*"},
	}
	fmt.Printf("Creating github webhook for %s/%s for url %s\n", owner, repo, webhookUrl)
	_, _, err = p.Client.Repositories.CreateHook(p.Context, owner, repo, hook)
	return err
}

func (p *GitHubProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	owner := data.Owner
	repo := data.Repo
	title := data.Title
	body := data.Body
	head := data.Head
	base := data.Base
	config := &github.NewPullRequest{}
	if title != "" {
		config.Title = github.String(title)
	}
	if body != "" {
		config.Body = github.String(body)
	}
	if head != "" {
		config.Head = github.String(head)
	}
	if base != "" {
		config.Base = github.String(base)
	}
	pr, _, err := p.Client.PullRequests.Create(p.Context, owner, repo, config)
	if err != nil {
		return nil, err
	}
	return &GitPullRequest{
		URL:    notNullString(pr.HTMLURL),
		Owner:  owner,
		Repo:   repo,
		Number: pr.Number,
	}, nil
}

func (p *GitHubProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for GitPullRequest %#v", pr)
	}
	n := *pr.Number
	result, _, err := p.Client.PullRequests.Get(p.Context, pr.Owner, pr.Repo, n)
	if err != nil {
		return err
	}
	head := result.Head
	if head != nil {
		pr.LastCommitSha = notNullString(head.SHA)
	} else {
		pr.LastCommitSha = ""
	}
	if result.Mergeable != nil {
		pr.Mergeable = result.Mergeable
	}
	pr.MergeCommitSHA = result.MergeCommitSHA
	if result.Merged != nil {
		pr.Merged = result.Merged
	}
	if result.ClosedAt != nil {
		pr.ClosedAt = result.ClosedAt
	}
	if result.MergedAt != nil {
		pr.MergedAt = result.MergedAt
	}
	if result.State != nil {
		pr.State = result.State
	}
	if result.StatusesURL != nil {
		pr.StatusesURL = result.StatusesURL
	}
	if result.IssueURL != nil {
		pr.IssueURL = result.IssueURL
	}
	if result.DiffURL != nil {
		pr.IssueURL = result.DiffURL
	}
	return nil
}

func (p *GitHubProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for GitPullRequest %#v", pr)
	}
	n := *pr.Number
	ref := pr.LastCommitSha
	options := &github.PullRequestOptions{
		SHA: ref,
	}
	result, _, err := p.Client.PullRequests.Merge(p.Context, pr.Owner, pr.Repo, n, message, options)
	if err != nil {
		return err
	}
	if result.Merged == nil || *result.Merged == false {
		return fmt.Errorf("Failed to merge PR %s for ref %s as result did not return merged", pr.URL)
	}
	return nil
}

func (p *GitHubProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for GitPullRequest %#v", pr)
	}
	n := *pr.Number

	prComment := &github.IssueComment{
		Body: &comment,
	}
	_, _, err := p.Client.Issues.CreateComment(p.Context, pr.Owner, pr.Repo, n, prComment)
	if err != nil {
		return err
	}
	return nil
}

func (p *GitHubProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	ref := pr.LastCommitSha
	if ref == "" {
		return "", fmt.Errorf("Missing String for LastCommitSha %#v", pr)
	}
	results, _, err := p.Client.Repositories.ListStatuses(p.Context, pr.Owner, pr.Repo, ref, nil)
	if err != nil {
		return "", err
	}
	for _, result := range results {
		if result.State != nil {
			return *result.State, nil
		}
	}
	return "", fmt.Errorf("Could not find a status for repository %s/%s with ref %s", pr.Owner, pr.Repo, ref)
}

func (p *GitHubProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	answer := []*GitRepoStatus{}
	results, _, err := p.Client.Repositories.ListStatuses(p.Context, org, repo, sha, nil)
	if err != nil {
		return answer, fmt.Errorf("Could not find a status for repository %s/%s with ref %s", org, repo, sha)
	}
	for _, result := range results {
		status := &GitRepoStatus{
			ID:          notNullInt64(result.ID),
			Context:     notNullString(result.Context),
			URL:         notNullString(result.URL),
			TargetURL:   notNullString(result.TargetURL),
			State:       notNullString(result.State),
			Description: notNullString(result.Description),
		}
		answer = append(answer, status)
	}
	return answer, nil
}

func notNullInt64(n *int64) int64 {
	if n != nil {
		return *n
	}
	return 0
}

func notNullString(tp *string) string {
	if tp == nil {
		return ""
	}
	return *tp
}

func (p *GitHubProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	if org == "" {
		org = p.Username
	}
	config := &github.Repository{
		Name: github.String(newName),
	}
	repo, _, err := p.Client.Repositories.Edit(p.Context, org, name, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to edit repository %s/%s due to: %s", org, name, err)
	}
	answer := &GitRepository{
		Name:             name,
		AllowMergeCommit: asBool(repo.AllowMergeCommit),
		CloneURL:         asText(repo.CloneURL),
		HTMLURL:          asText(repo.HTMLURL),
		SSHURL:           asText(repo.SSHURL),
	}
	return answer, nil
}

func (p *GitHubProvider) ValidateRepositoryName(org string, name string) error {
	_, r, err := p.Client.Repositories.Get(p.Context, org, name)
	if err == nil {
		return fmt.Errorf("Repository %s already exists", GitRepoName(org, name))
	}
	if r.StatusCode == 404 {
		return nil
	}
	return err
}

func (p *GitHubProvider) IsGitHub() bool {
	return true
}

func (p *GitHubProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/github-webhook/"
}

func GitHubAccessTokenURL(url string) string {
	return fmt.Sprintf("https://%s/settings/tokens/new?scopes=repo,read:user,user:email,write:repo_hook", url)
}

func (p *GitHubProvider) Label() string {
	return p.Server.Label()
}

func asBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func asText(text *string) string {
	if text != nil {
		return *text
	}
	return ""
}
