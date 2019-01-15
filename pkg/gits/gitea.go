package gits

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

type GiteaProvider struct {
	Username string
	Client   *gitea.Client

	Server auth.AuthServer
	User   auth.UserAuth
	Git    Gitter
}

func NewGiteaProvider(server *auth.AuthServer, user *auth.UserAuth, git Gitter) (GitProvider, error) {
	client := gitea.NewClient(server.URL, user.ApiToken)

	provider := GiteaProvider{
		Client:   client,
		Server:   *server,
		User:     *user,
		Username: user.Username,
		Git:      git,
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

func (p *GiteaProvider) ListReleases(org string, name string) ([]*GitRelease, error) {
	owner := org
	if owner == "" {
		owner = p.Username
	}
	answer := []*GitRelease{}
	repos, err := p.Client.ListReleases(owner, name)
	if err != nil {
		return answer, err
	}
	for _, repo := range repos {
		answer = append(answer, toGiteaRelease(org, name, repo))
	}
	return answer, nil
}

func toGiteaRelease(org string, name string, release *gitea.Release) *GitRelease {
	totalDownloadCount := 0
	assets := make([]GitReleaseAsset, 0)
	for _, asset := range release.Attachments {
		totalDownloadCount = totalDownloadCount + int(asset.DownloadCount)
		assets = append(assets, GitReleaseAsset{
			Name:               asset.Name,
			BrowserDownloadURL: asset.DownloadURL,
		})
	}
	return &GitRelease{
		Name:          release.Title,
		TagName:       release.TagName,
		Body:          release.Note,
		URL:           release.URL,
		HTMLURL:       release.URL,
		DownloadCount: totalDownloadCount,
		Assets:        &assets,
	}
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
			log.Warnf("Waiting for the fork of %s/%s to appear...\n", owner, name)
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
	repo := data.Repo.Name
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
			log.Warnf("Already has a webhook registered for %s\n", webhookUrl)
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
		Events: []string{"create", "push", "pull_request"},
		Active: true,
	}
	log.Infof("Creating Gitea webhook for %s/%s for url %s\n", util.ColorInfo(owner), util.ColorInfo(repo), util.ColorInfo(webhookUrl))
	_, err = p.Client.CreateRepoHook(owner, repo, hook)
	if err != nil {
		return fmt.Errorf("Failed to create webhook for %s/%s with %#v due to: %s", owner, repo, hook, err)
	}
	return err
}

func (p *GiteaProvider) ListWebHooks(owner string, repo string) ([]*GitWebHookArguments, error) {
	webHooks := []*GitWebHookArguments{}
	return webHooks, fmt.Errorf("not implemented!")
}

func (p *GiteaProvider) UpdateWebHook(data *GitWebHookArguments) error {
	return fmt.Errorf("not implemented!")
}

func (p *GiteaProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	owner := data.GitRepository.Organisation
	repo := data.GitRepository.Name
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
	id := int(pr.Index)
	answer := &GitPullRequest{
		URL:    pr.HTMLURL,
		Number: &id,
		Owner:  data.GitRepository.Organisation,
		Repo:   data.GitRepository.Name,
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
		return fmt.Errorf("Could not find pull request for %s/%s #%d: %s", pr.Owner, pr.Repo, n, err)
	}
	pr.Author = &GitUser{
		Login: result.Poster.UserName,
	}
	merged := result.HasMerged
	pr.Merged = &merged
	pr.Mergeable = &result.Mergeable
	pr.MergedAt = result.Merged
	pr.MergeCommitSHA = result.MergedCommitID
	pr.Title = result.Title
	pr.Body = result.Body
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

func (p *GiteaProvider) GetPullRequest(owner string, repo *GitRepository, number int) (*GitPullRequest, error) {
	pr := &GitPullRequest{
		Owner:  owner,
		Repo:   repo.Name,
		Number: &number,
	}
	err := p.UpdatePullRequestStatus(pr)
	return pr, err
}

func (p *GiteaProvider) GetPullRequestCommits(owner string, repository *GitRepository, number int) ([]*GitCommit, error) {
	answer := []*GitCommit{}

	// TODO there does not seem to be any way to get a diff of commits
	// unless maybe checking out the repo (do we have access to a local copy?)
	// there is a pr.Base and pr.Head that might be able to compare to get
	// commits somehow, but does not look like anything through the api

	return answer, nil
}

func (p *GiteaProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	i, err := p.Client.GetIssue(org, name, int64(number))
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	return p.fromGiteaIssue(org, name, i)
}

func (p *GiteaProvider) IssueURL(org string, name string, number int, isPull bool) string {
	serverPrefix := p.Server.URL
	if strings.Index(serverPrefix, "://") < 0 {
		serverPrefix = "https://" + serverPrefix
	}
	path := "issues"
	if isPull {
		path = "pull"
	}
	url := util.UrlJoin(serverPrefix, org, name, path, strconv.Itoa(number))
	return url
}

func (p *GiteaProvider) SearchIssues(org string, name string, filter string) ([]*GitIssue, error) {
	opts := gitea.ListIssueOption{}
	// TODO apply the filter?
	return p.searchIssuesWithOptions(org, name, opts)
}

func (p *GiteaProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	opts := gitea.ListIssueOption{}
	issues, err := p.searchIssuesWithOptions(org, name, opts)
	if err != nil {
		return issues, err
	}
	return FilterIssuesClosedSince(issues, t), nil
}

func (p *GiteaProvider) searchIssuesWithOptions(org string, name string, opts gitea.ListIssueOption) ([]*GitIssue, error) {
	opts.Page = 0
	answer := []*GitIssue{}
	issues, err := p.Client.ListRepoIssues(org, name, opts)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return answer, nil
		}
		return answer, err
	}
	for _, issue := range issues {
		i, err := p.fromGiteaIssue(org, name, issue)
		if err != nil {
			return answer, err
		}
		answer = append(answer, i)
	}
	return answer, nil
}

func (p *GiteaProvider) fromGiteaIssue(org string, name string, i *gitea.Issue) (*GitIssue, error) {
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
		URL:           p.IssueURL(org, name, number, false),
		State:         &state,
		Title:         i.Title,
		Body:          i.Body,
		IsPullRequest: i.PullRequest != nil,
		Labels:        labels,
		User:          toGiteaUser(i.Poster),
		Assignees:     assignees,
		CreatedAt:     &i.Created,
		UpdatedAt:     &i.Updated,
		ClosedAt:      i.Closed,
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
	return p.fromGiteaIssue(owner, repo, i)
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
			ID:          string(result.ID),
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

func (b *GiteaProvider) UpdateCommitStatus(org string, repo string, sha string, status *GitRepoStatus) (*GitRepoStatus, error) {
	return &GitRepoStatus{}, errors.New("TODO")
}

func (p *GiteaProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	return nil, fmt.Errorf("Rename of repositories is not supported for Gitea")
}

func (p *GiteaProvider) ValidateRepositoryName(org string, name string) error {
	_, err := p.Client.GetRepo(org, name)
	if err == nil {
		return fmt.Errorf("Repository %s already exists", p.Git.RepoName(org, name))
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

func (p *GiteaProvider) IsBitbucketCloud() bool {
	return false
}

func (p *GiteaProvider) IsBitbucketServer() bool {
	return false
}

func (p *GiteaProvider) IsGerrit() bool {
	return false
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

func (p *GiteaProvider) ServerURL() string {
	return p.Server.URL
}

func (p *GiteaProvider) BranchArchiveURL(org string, name string, branch string) string {
	return util.UrlJoin(p.ServerURL(), org, name, "archive", branch+".zip")
}

func (p *GiteaProvider) UserAuth() auth.UserAuth {
	return p.User
}

func (p *GiteaProvider) CurrentUsername() string {
	return p.Username
}

func (p *GiteaProvider) UserInfo(username string) *GitUser {
	user, err := p.Client.GetUserInfo(username)

	if err != nil {
		return nil
	}

	return &GitUser{
		Login:     username,
		Name:      user.FullName,
		AvatarURL: user.AvatarURL,
		Email:     user.Email,
		// TODO figure the Gitea user url
		URL: p.Server.URL + "/" + username,
	}
}

func (p *GiteaProvider) AddCollaborator(user string, organisation string, repo string) error {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for Gitea. Please add user: %v as a collaborator to this project.\n", user)
	return nil
}

func (p *GiteaProvider) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for Gitea.\n")
	return []*github.RepositoryInvitation{}, &github.Response{}, nil
}

func (p *GiteaProvider) AcceptInvitation(ID int64) (*github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for Gitea.\n")
	return &github.Response{}, nil
}

func (p *GiteaProvider) GetContent(org string, name string, path string, ref string) (*GitFileContent, error) {
	return nil, fmt.Errorf("Getting content not supported on gitea")
}
