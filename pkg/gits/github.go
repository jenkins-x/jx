package gits

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"golang.org/x/oauth2"
)

const (
	pageSize = 100
)

type GitHubProvider struct {
	Username string
	Client   *github.Client
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
	Git    Gitter
}

func NewGitHubProvider(server *auth.AuthServer, user *auth.UserAuth, git Gitter) (GitProvider, error) {
	ctx := context.Background()

	provider := GitHubProvider{
		Server:   *server,
		User:     *user,
		Context:  ctx,
		Username: user.Username,
		Git:      git,
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: user.ApiToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	var err error
	u := server.URL
	if IsGitHubServerURL(u) {
		provider.Client = github.NewClient(tc)
	} else {
		u = GitHubEnterpriseApiEndpointURL(u)
		provider.Client, err = github.NewEnterpriseClient(u, u, tc)
	}
	return &provider, err
}

func GitHubEnterpriseApiEndpointURL(u string) string {
	if IsGitHubServerURL(u) {
		return u
	}
	// lets ensure we use the API endpoint to login
	if strings.Index(u, "/api/") < 0 {
		u = util.UrlJoin(u, "/api/v3/")
	}
	return u
}

// GetEnterpriseApiURL returns the github enterprise API URL or blank if this
// provider is for the https://github.com service
func (p *GitHubProvider) GetEnterpriseApiURL() string {
	u := p.Server.URL
	if IsGitHubServerURL(u) {
		return ""
	}
	return GitHubEnterpriseApiEndpointURL(u)
}

func IsGitHubServerURL(u string) bool {
	u = strings.TrimSuffix(u, "/")
	return u == "" || u == "https://github.com" || u == "http://github.com"
}

func (p *GitHubProvider) ListOrganisations() ([]GitOrganisation, error) {
	answer := []GitOrganisation{}
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

func (p *GitHubProvider) IsUserInOrganisation(user string, org string) (bool, error) {
	membership, _, err := p.Client.Organizations.GetOrgMembership(p.Context, user, org)
	if err != nil {
		return false, err
	}
	if membership != nil {
		return true, nil
	}
	return false, nil
}

func (p *GitHubProvider) ListRepositories(org string) ([]*GitRepository, error) {
	owner := org
	answer := []*GitRepository{}
	options := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: pageSize,
		},
	}
	for {
		repos, _, err := p.Client.Repositories.ListByOrg(p.Context, owner, options)
		if err != nil {
			options := &github.RepositoryListOptions{
				ListOptions: github.ListOptions{
					Page:    0,
					PerPage: pageSize,
				},
			}
			repos, _, err = p.Client.Repositories.List(p.Context, owner, options)
			if err != nil {
				return answer, err
			}

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

func (p *GitHubProvider) ListReleases(org string, name string) ([]*GitRelease, error) {
	owner := org
	if owner == "" {
		owner = p.Username
	}
	answer := []*GitRelease{}
	options := &github.ListOptions{
		Page:    0,
		PerPage: pageSize,
	}
	for {
		repos, _, err := p.Client.Repositories.ListReleases(p.Context, owner, name, options)
		if err != nil {
			return answer, err
		}
		for _, repo := range repos {
			answer = append(answer, toGitHubRelease(org, name, repo))
		}
		if len(repos) < pageSize || len(repos) == 0 {
			break
		}
		options.Page += 1
	}
	return answer, nil
}

func toGitHubRelease(org string, name string, release *github.RepositoryRelease) *GitRelease {
	totalDownloadCount := 0
	assets := make([]GitReleaseAsset, 0)
	for _, asset := range release.Assets {
		p := asset.DownloadCount
		if p != nil {
			totalDownloadCount = totalDownloadCount + *p
		}
		assets = append(assets, GitReleaseAsset{
			Name:               asText(asset.Name),
			BrowserDownloadURL: asText(asset.BrowserDownloadURL),
			ContentType:        asText(asset.ContentType),
		})
	}
	return &GitRelease{
		Name:          asText(release.Name),
		TagName:       asText(release.TagName),
		Body:          asText(release.Body),
		URL:           asText(release.URL),
		HTMLURL:       asText(release.HTMLURL),
		DownloadCount: totalDownloadCount,
		Assets:        &assets,
	}
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
	if org == p.Username {
		org = ""
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
		Fork:             asBool(repo.Fork),
		Language:         asText(repo.Language),
		Stars:            asInt(repo.StargazersCount),
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
			log.Warnf("Waiting for the fork of %s/%s to appear...\n", owner, name)
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
	repo := data.Repo.Name
	if repo == "" {
		return fmt.Errorf("Missing property Repo")
	}
	webhookUrl := data.URL
	if webhookUrl == "" {
		return fmt.Errorf("Missing property URL")
	}
	hooks, _, err := p.Client.Repositories.ListHooks(p.Context, owner, repo, nil)
	if err != nil {
		log.Errorf("Error querying webhooks on %s/%s: %s\n", owner, repo, err)
	}
	for _, hook := range hooks {
		c := hook.Config["url"]
		s, ok := c.(string)
		if ok && s == webhookUrl {
			log.Warnf("Already has a webhook registered for %s\n", webhookUrl)
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
	log.Infof("Creating GitHub webhook for %s/%s for url %s\n", util.ColorInfo(owner), util.ColorInfo(repo), util.ColorInfo(webhookUrl))
	_, _, err = p.Client.Repositories.CreateHook(p.Context, owner, repo, hook)
	return err
}

func (p *GitHubProvider) ListWebHooks(owner string, repo string) ([]*GitWebHookArguments, error) {
	webHooks := []*GitWebHookArguments{}

	if owner == "" {
		owner = p.Username
	}
	if repo == "" {
		return webHooks, fmt.Errorf("Missing property Repo")
	}

	hooks, _, err := p.Client.Repositories.ListHooks(p.Context, owner, repo, nil)
	if err != nil {
		log.Errorf("Error querying webhooks on %s/%s: %s\n", owner, repo, err)
	}

	for _, hook := range hooks {
		c := hook.Config["url"]
		s, ok := c.(string)
		if ok {
			webHook := &GitWebHookArguments{
				ID:    hook.GetID(),
				Owner: owner,
				Repo:  nil,
				URL:   s,
			}
			webHooks = append(webHooks, webHook)
		}
	}

	return webHooks, nil
}

func (p *GitHubProvider) UpdateWebHook(data *GitWebHookArguments) error {
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
	hooks, _, err := p.Client.Repositories.ListHooks(p.Context, owner, repo, nil)
	if err != nil {
		log.Errorf("Error querying webhooks on %s/%s: %s\n", owner, repo, err)
	}

	dataId := data.ID
	if dataId == 0 {
		for _, hook := range hooks {
			c := hook.Config["url"]
			s, ok := c.(string)
			if ok && s == data.ExistingURL {
				log.Warnf("Found existing webhook for url %s\n", data.ExistingURL)
				dataId = hook.GetID()
			}
		}
	}

	if dataId != 0 {
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

		log.Infof("Updating GitHub webhook for %s/%s for url %s\n", util.ColorInfo(owner), util.ColorInfo(repo), util.ColorInfo(webhookUrl))
		_, _, err = p.Client.Repositories.EditHook(p.Context, owner, repo, dataId, hook)
	} else {
		log.Warn("No webhooks found to update")
	}
	return err
}

func (p *GitHubProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	owner := data.GitRepository.Organisation
	repo := data.GitRepository.Name
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
	if pr.Author == nil && result.User != nil && result.User.Login != nil {
		pr.Author = &GitUser{
			Login: *result.User.Login,
		}
	}
	pr.Assignees = make([]*GitUser, 0)
	for _, u := range result.Assignees {
		if u != nil {
			pr.Assignees = append(pr.Assignees, &GitUser{
				Login: *u.Login,
			})
		}
	}
	pr.RequestedReviewers = make([]*GitUser, 0)
	for _, u := range result.RequestedReviewers {
		if u != nil {
			pr.RequestedReviewers = append(pr.RequestedReviewers, &GitUser{
				Login: *u.Login,
			})
		}
	}
	pr.Labels = make([]*Label, 0)
	for _, l := range result.Labels {
		if l != nil {
			pr.Labels = append(pr.Labels, &Label{
				Name:        l.Name,
				URL:         l.URL,
				ID:          l.ID,
				Color:       l.Color,
				Default:     l.Default,
				Description: l.Description,
			})
		}
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
	if result.Head != nil {
		pr.HeadRef = result.Head.Ref
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
	if result.Title != nil {
		pr.Title = *result.Title
	}
	if result.Body != nil {
		pr.Body = *result.Body
	}
	if result.HTMLURL != nil {
		pr.URL = *result.HTMLURL
	}
	if result.UpdatedAt != nil {
		pr.UpdatedAt = result.UpdatedAt
	}
	return nil
}

func (p *GitHubProvider) GetPullRequest(owner string, repo *GitRepository, number int) (*GitPullRequest, error) {
	pr := &GitPullRequest{
		Owner:  owner,
		Repo:   repo.Name,
		Number: &number,
	}
	err := p.UpdatePullRequestStatus(pr)
	return pr, err
}

func (p *GitHubProvider) GetPullRequestCommits(owner string, repository *GitRepository, number int) ([]*GitCommit, error) {
	repo := repository.Name
	commits, _, err := p.Client.PullRequests.ListCommits(p.Context, owner, repo, number, nil)

	if err != nil {
		return nil, err
	}

	answer := []*GitCommit{}

	for _, commit := range commits {
		message := ""
		if commit.Commit != nil {
			message = commit.Commit.GetMessage()
			if commit.Author != nil {
				author := commit.Author

				summary := &GitCommit{
					Message: message,
					URL:     commit.GetURL(),
					SHA:     commit.GetSHA(),
					Author: &GitUser{
						Login:     author.GetLogin(),
						Email:     author.GetEmail(),
						Name:      author.GetName(),
						URL:       author.GetURL(),
						AvatarURL: author.GetAvatarURL(),
					},
				}

				if summary.Author.Email == "" {
					log.Info("Commit author email is empty for: " + commit.GetSHA() + "\n")
					dir, err := os.Getwd()
					if err != nil {
						return answer, err
					}
					gitDir, _, err := p.Git.FindGitConfigDir(dir)
					if err != nil {
						return answer, err
					}
					log.Info("Looking for commits in: " + gitDir + "\n")
					email, err := p.Git.GetAuthorEmailForCommit(gitDir, commit.GetSHA())
					if err != nil {
						log.Warn("Commit not found: " + commit.GetSHA() + "\n")
						continue
					}
					summary.Author.Email = email
				}

				answer = append(answer, summary)
			} else {
				log.Warn("No author for commit: " + commit.GetSHA() + "\n")
			}
		} else {
			log.Warn("No Commit object for for commit: " + commit.GetSHA() + "\n")
		}
	}
	return answer, nil
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
		return fmt.Errorf("Failed to merge PR %s for ref %s as result did not return merged", pr.URL, ref)
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

func (p *GitHubProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	issueComment := &github.IssueComment{
		Body: &comment,
	}
	_, _, err := p.Client.Issues.CreateComment(p.Context, owner, repo, number, issueComment)
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
	if sha == "" {
		return answer, fmt.Errorf("Missing String for sha %s/%s", org, repo)
	}
	results, _, err := p.Client.Repositories.ListStatuses(p.Context, org, repo, sha, nil)
	if err != nil {
		return answer, fmt.Errorf("Could not find a status for repository %s/%s with ref %s", org, repo, sha)
	}
	for _, result := range results {
		status := &GitRepoStatus{
			ID:          strconv.FormatInt(notNullInt64(result.ID), 10),
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

func (p *GitHubProvider) UpdateCommitStatus(org string, repo string, sha string, status *GitRepoStatus) (*GitRepoStatus, error) {
	id64 := int64(0)
	if status.ID != "" {
		id, err := strconv.Atoi(status.ID)
		if err != nil {
			return &GitRepoStatus{}, err
		}
		id64 = int64(id)
	}
	repoStatus := github.RepoStatus{
		Context:     &status.Context,
		State:       &status.State,
		Description: &status.Description,
		TargetURL:   &status.TargetURL,
		URL:         &status.URL,
		ID:          &id64,
	}
	result, _, err := p.Client.Repositories.CreateStatus(p.Context, org, repo, sha, &repoStatus)
	if err != nil {
		return &GitRepoStatus{}, err
	}
	return &GitRepoStatus{
		ID:          strconv.FormatInt(notNullInt64(result.ID), 10),
		Context:     notNullString(result.Context),
		URL:         notNullString(result.URL),
		TargetURL:   notNullString(result.TargetURL),
		State:       notNullString(result.State),
		Description: notNullString(result.Description),
	}, nil
}

func (p *GitHubProvider) GetContent(org string, name string, path string, ref string) (*GitFileContent, error) {
	fileContent, _, _, err := p.Client.Repositories.GetContents(p.Context, org, name, path, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, err
	}
	if fileContent != nil {
		return &GitFileContent{
			Name:        notNullString(fileContent.Name),
			Url:         notNullString(fileContent.URL),
			Path:        notNullString(fileContent.Path),
			Type:        notNullString(fileContent.Type),
			Content:     notNullString(fileContent.Content),
			DownloadUrl: notNullString(fileContent.DownloadURL),
			Encoding:    notNullString(fileContent.Encoding),
			GitUrl:      notNullString(fileContent.GitURL),
			HtmlUrl:     notNullString(fileContent.HTMLURL),
			Sha:         notNullString(fileContent.SHA),
			Size:        notNullInt(fileContent.Size),
		}, nil
	} else {
		return nil, fmt.Errorf("Directory Content not yet supported")
	}
}

func notNullInt64(n *int64) int64 {
	if n != nil {
		return *n
	}
	return 0
}

func notNullInt(n *int) int {
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
		return fmt.Errorf("Repository %s already exists", p.Git.RepoName(org, name))
	}
	if r != nil && r.StatusCode == 404 {
		return nil
	}
	return err
}

func (p *GitHubProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	release := &github.RepositoryRelease{}
	rel, r, err := p.Client.Repositories.GetReleaseByTag(p.Context, owner, repo, tag)

	if r != nil && r.StatusCode == 404 && !strings.HasPrefix(tag, "v") {
		// sometimes we prepend a v for example when using gh-release
		// so lets make sure we don't create a double release
		vtag := "v" + tag

		rel2, r2, err2 := p.Client.Repositories.GetReleaseByTag(p.Context, owner, repo, vtag)
		if r2.StatusCode != 404 {
			rel = rel2
			r = r2
			err = err2
			tag = vtag
		}
	}

	if r != nil && err == nil {
		release = rel
	}
	// lets populate the release
	if release.Name == nil && releaseInfo.Name != "" {
		release.Name = &releaseInfo.Name
	}
	if release.TagName == nil && releaseInfo.TagName != "" {
		release.TagName = &releaseInfo.TagName
	}
	if release.Body == nil && releaseInfo.Body != "" {
		release.Body = &releaseInfo.Body
	}
	if r != nil && r.StatusCode == 404 {
		log.Warnf("No release found for %s/%s and tag %s so creating a new release\n", owner, repo, tag)
		_, _, err = p.Client.Repositories.CreateRelease(p.Context, owner, repo, release)
		return err
	}
	id := release.ID
	if id == nil {
		return fmt.Errorf("The release for %s/%s tag %s has no ID!", owner, repo, tag)
	}
	r2, _, err := p.Client.Repositories.EditRelease(p.Context, owner, repo, *id, release)
	if r != nil {
		releaseInfo.URL = asText(r2.URL)
		releaseInfo.HTMLURL = asText(r2.HTMLURL)
	}
	return err
}

func (p *GitHubProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	i, r, err := p.Client.Issues.Get(p.Context, org, name, number)
	if r != nil && r.StatusCode == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p.fromGithubIssue(org, name, number, i)
}

func (p *GitHubProvider) SearchIssues(org string, name string, filter string) ([]*GitIssue, error) {
	opts := &github.IssueListByRepoOptions{}
	return p.searchIssuesWithOptions(org, name, opts)
}

func (p *GitHubProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	opts := &github.IssueListByRepoOptions{
		State: "closed",
	}
	issues, err := p.searchIssuesWithOptions(org, name, opts)
	if err != nil {
		return issues, err
	}
	issues = FilterIssuesClosedSince(issues, t)
	return issues, nil
}

func (p *GitHubProvider) searchIssuesWithOptions(org string, name string, opts *github.IssueListByRepoOptions) ([]*GitIssue, error) {
	opts.Page = 0
	opts.PerPage = pageSize
	answer := []*GitIssue{}
	for {
		issues, r, err := p.Client.Issues.ListByRepo(p.Context, org, name, opts)
		if r != nil && r.StatusCode == 404 {
			return answer, nil
		}
		if err != nil {
			return answer, err
		}
		for _, issue := range issues {
			if issue.Number != nil && !issue.IsPullRequest() {
				n := *issue.Number
				i, err := p.fromGithubIssue(org, name, n, issue)
				if err != nil {
					return answer, err
				}

				// TODO apply the filter?
				answer = append(answer, i)
			}
		}
		if len(issues) < pageSize || len(issues) == 0 {
			break
		}
		opts.ListOptions.Page += 1
	}
	return answer, nil
}

func (p *GitHubProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	labels := []string{}
	for _, label := range issue.Labels {
		name := label.Name
		if name != "" {
			labels = append(labels, name)
		}
	}
	config := &github.IssueRequest{
		Title:  &issue.Title,
		Body:   &issue.Body,
		Labels: &labels,
	}
	i, _, err := p.Client.Issues.Create(p.Context, owner, repo, config)
	if err != nil {
		return nil, err
	}
	number := 0
	if i.Number != nil {
		number = *i.Number
	}
	return p.fromGithubIssue(owner, repo, number, i)
}

func (p *GitHubProvider) fromGithubIssue(org string, name string, number int, i *github.Issue) (*GitIssue, error) {
	isPull := i.IsPullRequest()
	url := p.IssueURL(org, name, number, isPull)

	labels := []GitLabel{}
	for _, label := range i.Labels {
		labels = append(labels, toGitHubLabel(&label))
	}
	assignees := []GitUser{}
	for _, assignee := range i.Assignees {
		assignees = append(assignees, *toGitHubUser(assignee))
	}
	return &GitIssue{
		Number:        &number,
		URL:           url,
		State:         i.State,
		Title:         asText(i.Title),
		Body:          asText(i.Body),
		IsPullRequest: isPull,
		Labels:        labels,
		User:          toGitHubUser(i.User),
		CreatedAt:     i.CreatedAt,
		UpdatedAt:     i.UpdatedAt,
		ClosedAt:      i.ClosedAt,
		ClosedBy:      toGitHubUser(i.ClosedBy),
		Assignees:     assignees,
	}, nil
}

func (p *GitHubProvider) IssueURL(org string, name string, number int, isPull bool) string {
	serverPrefix := p.Server.URL
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

func toGitHubUser(user *github.User) *GitUser {
	if user == nil {
		return nil
	}
	return &GitUser{
		Login:     asText(user.Login),
		Name:      asText(user.Name),
		Email:     asText(user.Email),
		AvatarURL: asText(user.AvatarURL),
	}
}

func toGitHubLabel(label *github.Label) GitLabel {
	return GitLabel{
		Name:  asText(label.Name),
		Color: asText(label.Color),
		URL:   asText(label.URL),
	}
}

func (p *GitHubProvider) HasIssues() bool {
	return true
}

func (p *GitHubProvider) IsGitHub() bool {
	return true
}

func (p *GitHubProvider) IsGitea() bool {
	return false
}

func (p *GitHubProvider) IsBitbucketCloud() bool {
	return false
}

func (p *GitHubProvider) IsBitbucketServer() bool {
	return false
}

func (p *GitHubProvider) IsGerrit() bool {
	return false
}

func (p *GitHubProvider) Kind() string {
	return KindGitHub
}

func (p *GitHubProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/github-webhook/"
}

func GitHubAccessTokenURL(url string) string {
	if strings.Index(url, "://") < 0 {
		url = "https://" + url
	}
	return util.UrlJoin(url, "/settings/tokens/new?scopes=repo,read:user,read:org,user:email,write:repo_hook,delete_repo")
}

func (p *GitHubProvider) Label() string {
	return p.Server.Label()
}

func (p *GitHubProvider) ServerURL() string {
	return p.Server.URL
}

func (p *GitHubProvider) BranchArchiveURL(org string, name string, branch string) string {
	return util.UrlJoin("https://codeload.github.com", org, name, "zip", branch)
}

func (p *GitHubProvider) CurrentUsername() string {
	return p.Username
}

func (p *GitHubProvider) UserAuth() auth.UserAuth {
	return p.User
}

func (p *GitHubProvider) UserInfo(username string) *GitUser {
	user, _, err := p.Client.Users.Get(p.Context, username)
	if user == nil || err != nil {
		log.Error("Unable to fetch user info for " + username + "\n")
		return nil
	}

	return &GitUser{
		Login:     username,
		Name:      user.GetName(),
		AvatarURL: user.GetAvatarURL(),
		URL:       user.GetHTMLURL(),
		Email:     user.GetEmail(),
	}
}

func (p *GitHubProvider) AddCollaborator(user string, organisation string, repo string) error {
	log.Infof("Automatically adding the pipeline user: %v as a collaborator.\n", user)
	_, err := p.Client.Repositories.AddCollaborator(p.Context, organisation, repo, user, &github.RepositoryAddCollaboratorOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (p *GitHubProvider) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	return p.Client.Users.ListInvitations(p.Context, &github.ListOptions{})
}

func (p *GitHubProvider) AcceptInvitation(ID int64) (*github.Response, error) {
	log.Infof("Automatically accepted invitation: %v for the pipeline user.\n", ID)
	return p.Client.Users.AcceptInvitation(p.Context, ID)
}

func asBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func asInt(i *int) int {
	if i != nil {
		return *i
	}
	return 0
}

func asText(text *string) string {
	if text != nil {
		return *text
	}
	return ""
}
