package gits

import (
	"fmt"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
)

type GiteaProvider struct {
	Username string
	Client   *gitea.Client

	Server auth.AuthServer
	User   auth.UserAuth
}

func NewGiteaProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	client := gitea.NewClient(server.URL, user.ApiToken)

	provider := GiteaProvider{
		Client:   client,
		Server:   *server,
		User:     *user,
		Username: user.Username,
	}

	return &provider, nil
}

func (p *GiteaProvider) ListOrganisations() ([]GitOrganisation, error) {
	answer := []GitOrganisation{}
	orgs, err := p.Client.ListMyOrgs()
	if err != nil {
		return answer, err
	}

	for _, org := range orgs {
		name := org.UserName
		if name != "" {
			o := GitOrganisation{
				Login: name,
			}
			answer = append(answer, o)
		}
	}
	return answer, nil
}

func (p *GiteaProvider) ListRepositories(org string) ([]*GitRepository, error) {
	answer := []*GitRepository{}
	if org == "" {
		repos, err := p.Client.ListMyRepos()
		if err != nil {
			return answer, err
		}
		for _, repo := range repos {
			answer = append(answer, toGiteaRepo(repo.Name, repo))
		}
		return answer, nil
	}
	repos, err := p.Client.ListOrgRepos(org)
	if err != nil {
		return answer, err
	}
	for _, repo := range repos {
		answer = append(answer, toGiteaRepo(repo.Name, repo))
	}
	return answer, nil
}

func (p *GiteaProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	options := gitea.CreateRepoOption{
		Name:    name,
		Private: private,
	}
	repo, err := p.Client.CreateRepo(options)
	if err != nil {
		return nil, fmt.Errorf("Failed to create repository %s/%s due to: %s", org, name, err)
	}
	return toGiteaRepo(name, repo), nil
}

func (p *GiteaProvider) GetRepository(org string, name string) (*GitRepository, error) {
	repo, err := p.Client.GetRepo(org, name)
	if err != nil {
		return nil, fmt.Errorf("Failed to get repository %s/%s due to: %s", org, name, err)
	}
	return toGiteaRepo(name, repo), nil
}

func (p *GiteaProvider) DeleteRepository(org string, name string) error {
	owner := org
	if owner == "" {
		owner = p.Username
	}
	err := p.Client.DeleteRepo(owner, name)
	if err != nil {
		return fmt.Errorf("Failed to delete repository %s/%s due to: %s", owner, name, err)
	}
	return err
}

func toGiteaRepo(name string, repo *gitea.Repository) *GitRepository {
	return &GitRepository{
		Name:             name,
		AllowMergeCommit: true,
		CloneURL:         repo.CloneURL,
		HTMLURL:          repo.HTMLURL,
		SSHURL:           repo.SSHURL,
		Fork:             repo.Fork,
	}
}

func (p *GiteaProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	repoConfig := gitea.CreateForkOption{
		Organization: &destinationOrg,
	}
	repo, err := p.Client.CreateFork(originalOrg, name, repoConfig)
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
				repo, err = p.Client.GetRepo(owner, name)
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
	return toGiteaRepo(name, repo), nil
}

func (p *GiteaProvider) CreateWebHook(data *GitWebHookArguments) error {
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
	hooks, err := p.Client.ListRepoHooks(owner, repo)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		s := hook.Config["url"]
		if s == webhookUrl {
			fmt.Printf("Already has a webhook registered for %s\n", webhookUrl)
			return nil
		}
	}
	config := map[string]string{
		"url":          webhookUrl,
		"content_type": "json",
	}
	if data.Secret != "" {
		config["secret"] = data.Secret
	}
	hook := gitea.CreateHookOption{
		Type:   "gitea",
		Config: config,
		Events: []string{"*"},
	}
	fmt.Printf("Creating github webhook for %s/%s for url %s\n", owner, repo, webhookUrl)
	_, err = p.Client.CreateRepoHook(owner, repo, hook)
	if err != nil {
		return fmt.Errorf("Failed to create webhook for %s/%s with %#v due to: %s", owner, repo, hook, err)
	}
	return err
}

func (p *GiteaProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	owner := data.Owner
	repo := data.Repo
	title := data.Title
	body := data.Body
	head := data.Head
	base := data.Base
	config := gitea.CreatePullRequestOption{}
	if title != "" {
		config.Title = title
	}
	if body != "" {
		config.Body = body
	}
	if head != "" {
		config.Head = head
	}
	if base != "" {
		config.Base = base
	}
	pr, err := p.Client.CreatePullRequest(owner, repo, config)
	if err != nil {
		return nil, err
	}
	id := int(pr.ID)
	answer := &GitPullRequest{
		URL:    pr.HTMLURL,
		Number: &id,
		Owner:  data.Owner,
		Repo:   data.Repo,
	}
	if pr.Head != nil {
		answer.LastCommitSha = pr.Head.Sha
	}
	return answer, nil
}

func (p *GiteaProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for GitPullRequest %#v", pr)
	}
	n := *pr.Number
	result, err := p.Client.GetPullRequest(pr.Owner, pr.Repo, int64(n))
	if err != nil {
		return err
	}
	merged := result.HasMerged
	pr.Merged = &merged
	pr.Mergeable = &result.Mergeable
	pr.MergedAt = result.Merged
	pr.MergeCommitSHA = result.MergedCommitID
	stateText := string(result.State)
	pr.State = &stateText
	head := result.Head
	if head != nil {
		pr.LastCommitSha = head.Sha
	} else {
		pr.LastCommitSha = ""
	}
	/*
		TODO

		pr.ClosedAt = result.Closed
		pr.StatusesURL = result.StatusesURL
		pr.IssueURL = result.IssueURL
		pr.DiffURL = result.DiffURL
	*/
	return nil
}

func (p *GiteaProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	i, err := p.Client.GetIssue(org, name, int64(number))
	if strings.Contains(err.Error(), "404") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p.fromGiteaIssue(i)
}

func (p *GiteaProvider) fromGiteaIssue(i *gitea.Issue) (*GitIssue, error) {
	state := string(i.State)
	labels := []GitLabel{}
	for _, label := range i.Labels {
		labels = append(labels, toGiteaLabel(label))
	}
	assignees := []GitUser{}
	assignee := i.Assignee
	if assignee != nil {
		assignees = append(assignees, *toGiteaUser(assignee))
	}
	number := int(i.ID)
	return &GitIssue{
		Number:        &number,
		URL:           i.URL,
		State:         &state,
		Title:         i.Title,
		Body:          i.Body,
		IsPullRequest: i.PullRequest != nil,
		Labels:        labels,
		User:          toGiteaUser(i.Poster),
		Assignees:     assignees,
	}, nil
}

func (p *GiteaProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	config := gitea.CreateIssueOption{
		Title: issue.Title,
		Body:  issue.Body,
	}
	i, err := p.Client.CreateIssue(owner, repo, config)
	if err != nil {
		return nil, err
	}
	return p.fromGiteaIssue(i)
}

func toGiteaLabel(label *gitea.Label) GitLabel {
	return GitLabel{
		Name:  label.Name,
		Color: label.Color,
		URL:   label.URL,
	}
}

func toGiteaUser(user *gitea.User) *GitUser {
	return &GitUser{
		Login:     user.UserName,
		Name:      user.FullName,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
	}
}

func (p *GiteaProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for GitPullRequest %#v", pr)
	}
	n := *pr.Number
	return p.Client.MergePullRequest(pr.Owner, pr.Repo, int64(n))
}

func (p *GiteaProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	ref := pr.LastCommitSha
	if ref == "" {
		return "", fmt.Errorf("Missing String for LastCommitSha %#v", pr)
	}
	results, err := p.Client.ListStatuses(pr.Owner, pr.Repo, ref, gitea.ListStatusesOption{})
	if err != nil {
		return "", err
	}
	for _, result := range results {
		text := string(result.State)
		if text != "" {
			return text, nil
		}
	}
	return "", fmt.Errorf("Could not find a status for repository %s/%s with ref %s", pr.Owner, pr.Repo, ref)
}

func (p *GiteaProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for GitPullRequest %#v", pr)
	}
	n := *pr.Number
	prComment := gitea.CreateIssueCommentOption{
		Body: asText(&comment),
	}
	_, err := p.Client.CreateIssueComment(pr.Owner, pr.Repo, int64(n), prComment)
	return err
}

func (p *GiteaProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	issueComment := gitea.CreateIssueCommentOption{
		Body: comment,
	}
	_, err := p.Client.CreateIssueComment(owner, repo, int64(number), issueComment)
	if err != nil {
		return err
	}
	return nil
}

func (p *GiteaProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	answer := []*GitRepoStatus{}
	results, err := p.Client.ListStatuses(org, repo, sha, gitea.ListStatusesOption{})
	if err != nil {
		return answer, fmt.Errorf("Could not find a status for repository %s/%s with ref %s", org, repo, sha)
	}
	for _, result := range results {
		status := &GitRepoStatus{
			ID:          result.ID,
			Context:     result.Context,
			URL:         result.URL,
			TargetURL:   result.TargetURL,
			State:       string(result.State),
			Description: result.Description,
		}
		answer = append(answer, status)
	}
	return answer, nil
}

func (p *GiteaProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	return nil, fmt.Errorf("Rename of repositories is not supported for gitea")
}

func (p *GiteaProvider) ValidateRepositoryName(org string, name string) error {
	_, err := p.Client.GetRepo(org, name)
	if err == nil {
		return fmt.Errorf("Repository %s already exists", GitRepoName(org, name))
	}
	if strings.Contains(err.Error(), "404") {
		return nil
	}
	return err
}

func (p *GiteaProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	var release *gitea.Release
	releases, err := p.Client.ListReleases(owner, repo)
	found := false
	for _, rel := range releases {
		if rel.TagName == tag {
			release = rel
			found = true
			break
		}
	}
	flag := false

	// lets populate the release
	if !found {
		createRelease := gitea.CreateReleaseOption{
			TagName:      releaseInfo.TagName,
			Title:        releaseInfo.Name,
			Note:         releaseInfo.Body,
			IsDraft:      flag,
			IsPrerelease: flag,
		}
		_, err = p.Client.CreateRelease(owner, repo, createRelease)
		return err
	} else {
		editRelease := gitea.EditReleaseOption{
			TagName:      release.TagName,
			Title:        release.Title,
			Note:         release.Note,
			IsDraft:      &flag,
			IsPrerelease: &flag,
		}
		if editRelease.Title == "" && releaseInfo.Name != "" {
			editRelease.Title = releaseInfo.Name
		}
		if editRelease.TagName == "" && releaseInfo.TagName != "" {
			editRelease.TagName = releaseInfo.TagName
		}
		if editRelease.Note == "" && releaseInfo.Body != "" {
			editRelease.Note = releaseInfo.Body
		}
		r2, err := p.Client.EditRelease(owner, repo, release.ID, editRelease)
		if err != nil {
			return err
		}
		if r2 != nil {
			releaseInfo.URL = r2.URL
		}
	}
	return err
}

func (p *GiteaProvider) HasIssues() bool {
	return true
}

func (p *GiteaProvider) IsGitHub() bool {
	return false
}

func (p *GiteaProvider) IsGitea() bool {
	return true
}

func (p *GiteaProvider) Kind() string {
	return "gitea"
}

func (p *GiteaProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/gitea-webhook/post"
}

func GiteaAccessTokenURL(url string) string {
	return util.UrlJoin(url, "/user/settings/applications")
}

func (p *GiteaProvider) Label() string {
	return p.Server.Label()
}
