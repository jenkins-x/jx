package gits

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/wbrefvem/go-bitbucket"
)

// BitbucketCloudProvider implements GitProvider interface for bitbucket.org
type BitbucketCloudProvider struct {
	Client   *bitbucket.APIClient
	Username string
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
}

var stateMap = map[string]string{
	"SUCCESSFUL": "success",
	"FAILED":     "failure",
	"INPROGRESS": "in-progress",
	"STOPPED":    "stopped",
}

func NewBitbucketCloudProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	ctx := context.Background()

	basicAuth := bitbucket.BasicAuth{
		UserName: user.Username,
		Password: user.ApiToken,
	}
	basicAuthContext := context.WithValue(ctx, bitbucket.ContextBasicAuth, basicAuth)

	provider := BitbucketCloudProvider{
		Server:   *server,
		User:     *user,
		Username: user.Username,
		Context:  basicAuthContext,
	}

	cfg := bitbucket.NewConfiguration()
	provider.Client = bitbucket.NewAPIClient(cfg)

	return &provider, nil
}

func (b *BitbucketCloudProvider) ListOrganisations() ([]GitOrganisation, error) {

	teams := []GitOrganisation{}

	// Pagination is gross.
	for {
		results, _, err := b.Client.TeamsApi.TeamsGet(b.Context, map[string]interface{}{"role": "member"})

		if err != nil {
			return nil, err
		}

		for _, team := range results.Values {
			teams = append(teams, GitOrganisation{Login: team.Username})
		}

		if results.Next == "" {
			break
		}
	}

	return teams, nil
}

func BitbucketRepositoryToGitRepository(bRepo bitbucket.Repository) *GitRepository {
	var sshURL string
	var httpCloneURL string
	for _, link := range bRepo.Links.Clone {
		if link.Name == "ssh" {
			sshURL = link.Href
		}
	}
	isFork := false
	if bRepo.Parent != nil {
		isFork = true
	}
	if httpCloneURL == "" {
		httpCloneURL = bRepo.Links.Html.Href
		if !strings.HasSuffix(httpCloneURL, ".git") {
			httpCloneURL += ".git"
		}
	}
	if httpCloneURL == "" {
		httpCloneURL = sshURL
	}
	return &GitRepository{
		Name:     bRepo.Name,
		HTMLURL:  bRepo.Links.Html.Href,
		CloneURL: httpCloneURL,
		SSHURL:   sshURL,
		Language: bRepo.Language,
		Fork:     isFork,
	}
}

func (b *BitbucketCloudProvider) ListRepositories(org string) ([]*GitRepository, error) {

	repos := []*GitRepository{}

	for {
		results, _, err := b.Client.RepositoriesApi.RepositoriesUsernameGet(b.Context, org, nil)

		if err != nil {
			return nil, err
		}

		for _, repo := range results.Values {
			repos = append(repos, BitbucketRepositoryToGitRepository(repo))
		}

		if results.Next == "" {
			break
		}
	}

	return repos, nil
}

func (b *BitbucketCloudProvider) CreateRepository(
	org string,
	name string,
	private bool,
) (*GitRepository, error) {

	options := map[string]interface{}{}
	options["body"] = bitbucket.Repository{
		IsPrivate: private,
	}

	result, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugPost(
		b.Context,
		b.Username,
		name,
		options,
	)

	if err != nil {
		return nil, err
	}

	return BitbucketRepositoryToGitRepository(result), nil
}

func (b *BitbucketCloudProvider) GetRepository(
	org string,
	name string,
) (*GitRepository, error) {

	repo, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugGet(
		b.Context,
		org,
		name,
	)

	if err != nil {
		return nil, err
	}

	return BitbucketRepositoryToGitRepository(repo), nil
}

func (b *BitbucketCloudProvider) DeleteRepository(org string, name string) error {

	_, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugDelete(
		b.Context,
		b.Username,
		name,
		nil,
	)

	if err != nil {
		return err
	}

	return nil
}

func (b *BitbucketCloudProvider) ForkRepository(
	originalOrg string,
	name string,
	destinationOrg string,
) (*GitRepository, error) {
	options := map[string]interface{}{
		"body": map[string]interface{}{},
	}
	repo, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugForksPost(
		b.Context,
		originalOrg,
		name,
		options,
	)

	if err != nil {
		return nil, err
	}

	_, _, err = b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugForksGet(
		b.Context,
		b.Username,
		repo.Name,
	)

	// Fork isn't ready
	if err != nil {

		// Wait up to 1 minute for the fork to be ready
		for i := 0; i < 30; i++ {
			_, _, err = b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugForksGet(
				b.Context,
				b.Username,
				repo.Name,
			)

			if err == nil {
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

	return BitbucketRepositoryToGitRepository(repo), nil
}

func (b *BitbucketCloudProvider) RenameRepository(
	org string,
	name string,
	newName string,
) (*GitRepository, error) {

	var options = map[string]interface{}{
		"name": newName,
	}

	repo, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugPut(
		b.Context,
		b.Username,
		name,
		options,
	)

	if err != nil {
		return nil, err
	}

	return BitbucketRepositoryToGitRepository(repo), nil
}

func (b *BitbucketCloudProvider) ValidateRepositoryName(org string, name string) error {

	_, r, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugGet(
		b.Context,
		b.Username,
		name,
	)

	if r != nil && r.StatusCode == 404 {
		return nil
	}

	if err == nil {
		return fmt.Errorf("repository %s/%s already exists", b.Username, name)
	}

	return err
}

func (b *BitbucketCloudProvider) CreatePullRequest(
	data *GitPullRequestArguments,
) (*GitPullRequest, error) {

	head := bitbucket.PullrequestEndpointBranch{Name: data.Head}
	sourceFullName := fmt.Sprintf("%s/%s", b.Username, data.GitRepositoryInfo.Name)
	sourceRepo := bitbucket.Repository{FullName: sourceFullName}
	source := bitbucket.PullrequestEndpoint{
		Repository: &sourceRepo,
		Branch:     &head,
	}

	base := bitbucket.PullrequestEndpointBranch{Name: data.Base}
	destination := bitbucket.PullrequestEndpoint{
		Branch: &base,
	}

	bPullrequest := bitbucket.Pullrequest{
		Source:      &source,
		Destination: &destination,
		Title:       data.Title,
	}

	var options = map[string]interface{}{
		"body": bPullrequest,
	}

	pr, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPost(
		b.Context,
		b.Username,
		data.GitRepositoryInfo.Name,
		options,
	)

	if err != nil {
		return nil, err
	}

	_, _, err = b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(
		b.Context,
		b.Username,
		data.GitRepositoryInfo.Name,
		pr.Id,
	)

	if err != nil {
		// Wait up to 1 minute for the PR to be ready.
		for i := 0; i < 30; i++ {
			_, _, err = b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(
				b.Context,
				b.Username,
				data.GitRepositoryInfo.Name,
				pr.Id,
			)

			if err == nil {
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

	i := int(pr.Id)
	prID := &i

	newPR := &GitPullRequest{
		URL:    pr.Links.Html.Href,
		Owner:  pr.Author.Username,
		Repo:   pr.Destination.Repository.FullName,
		Number: prID,
		State:  &pr.State,
	}

	return newPR, nil
}

func (b *BitbucketCloudProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {

	prID := int32(*pr.Number)
	bitbucketPR, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(
		b.Context,
		b.Username,
		strings.TrimPrefix(pr.Repo, pr.Owner+"/"),
		prID,
	)

	if err != nil {
		return err
	}

	pr.State = &bitbucketPR.State
	pr.Title = bitbucketPR.Title
	pr.Body = bitbucketPR.Summary.Raw
	pr.Author = &GitUser{
		Login: bitbucketPR.Author.Username,
	}

	if bitbucketPR.MergeCommit != nil {
		pr.MergeCommitSHA = &bitbucketPR.MergeCommit.Hash
	}
	pr.DiffURL = &bitbucketPR.Links.Diff.Href

	if bitbucketPR.State == "MERGED" {
		merged := true
		pr.Merged = &merged
	}

	commits, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet(
		b.Context,
		b.Username,
		strconv.FormatInt(int64(prID), 10),
		strings.TrimPrefix(pr.Repo, pr.Owner+"/"),
	)

	if err != nil {
		return err
	}

	values := commits["values"].([]interface{})
	commit := values[0].(map[string]interface{})

	pr.LastCommitSha = commit["hash"].(string)

	return nil
}

func (p *BitbucketCloudProvider) GetPullRequest(owner string, repoInfo *GitRepositoryInfo, number int) (*GitPullRequest, error) {
	repo := repoInfo.Name
	pr, _, err := p.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(
		p.Context,
		owner,
		repo,
		int32(number),
	)

	if err != nil {
		return nil, err
	}

	author := p.UserInfo(pr.Author.Username)

	if author.Email == "" {
		// bitbucket makes this part difficult, there is no way to directly
		// associate a username to an email through the api or vice versa
		// so our best attempt is to try to figure out the author email
		// from the commits
		commits, err := p.GetPullRequestCommits(owner, repoInfo, number)

		if err != nil {
			log.Warn("Unable to get commits for PR: " + owner + "/" + repo + "/" + strconv.Itoa(number) + " -- " + err.Error())
		}

		// we get correct login and email per commit, find the matching author
		for _, commit := range commits {
			if commit.Author.Login == author.Login {
				author.Email = commit.Author.Email
				break
			}
		}
	}

	return &GitPullRequest{
		URL:    pr.Links.Html.Href,
		Owner:  pr.Author.Username,
		Repo:   pr.Destination.Repository.FullName,
		Number: &number,
		State:  &pr.State,
		Author: author,
	}, nil
}

func (b *BitbucketCloudProvider) GetPullRequestCommits(owner string, repository *GitRepositoryInfo, number int) ([]*GitCommit, error) {
	repo := repository.Name
	answer := []*GitCommit{}

	// for some reason the 2nd parameter is the PR id, seems like an inconsistency/bug in the api
	commits, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet(b.Context, owner, strconv.Itoa(number), repo)
	if err != nil {
		return answer, err
	}

	commitVals, ok := commits["values"]
	if !ok {
		return answer, fmt.Errorf("No value key for %s/%s/%d", owner, repo, number)
	}

	commitValues, ok := commitVals.([]interface{})
	if !ok {
		return answer, fmt.Errorf("No commitValues for %s/%s/%d", owner, repo, number)
	}

	rawEmailMatcher, _ := regexp.Compile("[^<]*<([^>]+)>")

	for _, data := range commitValues {
		if data == nil {
			continue
		}

		comm, ok := data.(map[string]interface{})
		if !ok {
			log.Warn(fmt.Sprintf("Unexpected data structure for GetPullRequestCommits values from PR %s/%s/%d", owner, repo, number))
			continue
		}

		shaVal, ok := comm["hash"]
		if !ok {
			continue
		}

		sha, ok := shaVal.(string)
		if !ok {
			log.Warn(fmt.Sprintf("Unexpected data structure for GetPullRequestCommits hash from PR %s/%s/%d", owner, repo, number))
			continue
		}

		commit, _, err := b.Client.CommitsApi.RepositoriesUsernameRepoSlugCommitRevisionGet(b.Context, owner, repo, sha)
		if err != nil {
			return answer, err
		}

		url := ""
		if commit.Links != nil && commit.Links.Self != nil {
			url = commit.Links.Self.Href
		}

		// update the login and email
		login := ""
		email := ""
		if commit.Author != nil {
			// commit.Author is the actual Bitbucket user
			if commit.Author.User != nil {
				login = commit.Author.User.Username
			}
			// Author.Raw contains the Git commit author in the form: User <email@example.com>
			email = rawEmailMatcher.ReplaceAllString(commit.Author.Raw, "$1")
		}

		summary := &GitCommit{
			Message: commit.Message,
			URL:     url,
			SHA:     commit.Hash,
			Author: &GitUser{
				Login: login,
				Email: email,
			},
		}

		answer = append(answer, summary)
	}
	return answer, nil
}

func (b *BitbucketCloudProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {

	latestCommitStatus := bitbucket.Commitstatus{}

	for {
		result, _, err := b.Client.CommitstatusesApi.RepositoriesUsernameRepoSlugCommitNodeStatusesGet(
			b.Context,
			b.Username,
			strings.TrimPrefix(pr.Repo, pr.Owner+"/"),
			pr.LastCommitSha,
		)

		if err != nil {
			return "", err
		}

		// Our first time building, so return "success"
		if result.Size == 0 {
			return "success", nil
		}

		for _, status := range result.Values {
			if status.CreatedOn.After(latestCommitStatus.CreatedOn) {
				latestCommitStatus = status
			}
		}

		if result.Next == "" {
			break
		}
	}

	return stateMap[latestCommitStatus.State], nil
}

func (b *BitbucketCloudProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {

	statuses := []*GitRepoStatus{}

	for {
		result, _, err := b.Client.CommitstatusesApi.RepositoriesUsernameRepoSlugCommitNodeStatusesGet(
			b.Context,
			org,
			strings.TrimPrefix(repo, org+"/"),
			sha,
		)

		if err != nil {
			return nil, err
		}

		for _, status := range result.Values {

			if err != nil {
				return nil, err
			}

			newStatus := &GitRepoStatus{
				ID:          status.Key,
				URL:         status.Links.Commit.Href,
				State:       stateMap[status.State],
				TargetURL:   status.Links.Self.Href,
				Description: status.Description,
			}
			statuses = append(statuses, newStatus)
		}

		if result.Next == "" {
			break
		}
	}
	return statuses, nil
}

func (b *BitbucketCloudProvider) MergePullRequest(pr *GitPullRequest, message string) error {

	options := map[string]interface{}{
		"body": map[string]interface{}{
			"pullrequest_merge_parameters": map[string]interface{}{
				"message": message,
			},
		},
	}

	_, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdMergePost(
		b.Context,
		b.Username,
		strconv.FormatInt(int64(*pr.Number), 10),
		strings.TrimPrefix(pr.Repo, pr.Owner+"/"),
		options,
	)

	if err != nil {
		return err
	}

	return nil
}

func (b *BitbucketCloudProvider) CreateWebHook(data *GitWebHookArguments) error {

	options := map[string]interface{}{
		"body": map[string]interface{}{
			"url":    data.URL,
			"active": true,
			"events": []string{
				"repo:push",
			},
			"description": "Jenkins X Web Hook",
		},
	}

	_, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugHooksPost(
		b.Context,
		b.Username,
		data.Repo.Name,
		options,
	)

	if err != nil {
		return err
	}
	return nil
}

func BitbucketIssueToGitIssue(bIssue bitbucket.Issue) *GitIssue {
	id := int(bIssue.Id)
	ownerAndRepo := strings.Split(bIssue.Repository.FullName, "/")
	owner := ownerAndRepo[0]

	var assignee GitUser

	if bIssue.Assignee != nil {
		assignee = GitUser{
			URL:   bIssue.Assignee.Links.Self.Href,
			Login: bIssue.Assignee.Username,
			Name:  bIssue.Assignee.DisplayName,
		}
	}
	gitIssue := &GitIssue{
		URL:       bIssue.Links.Self.Href,
		Owner:     owner,
		Repo:      bIssue.Repository.Name,
		Number:    &id,
		Title:     bIssue.Title,
		Body:      bIssue.Content.Markup,
		State:     &bIssue.State,
		IssueURL:  &bIssue.Links.Html.Href,
		CreatedAt: &bIssue.CreatedOn,
		UpdatedAt: &bIssue.UpdatedOn,
		ClosedAt:  &bIssue.UpdatedOn,
		Assignees: []GitUser{
			assignee,
		},
	}
	return gitIssue
}

func (b *BitbucketCloudProvider) GitIssueToBitbucketIssue(gIssue GitIssue) bitbucket.Issue {

	bitbucketIssue := bitbucket.Issue{
		Title:      gIssue.Title,
		Repository: &bitbucket.Repository{Name: gIssue.Repo},
		Reporter:   &bitbucket.User{Username: b.Username},
	}

	return bitbucketIssue
}

func (b *BitbucketCloudProvider) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {

	gitIssues := []*GitIssue{}

	for {
		issues, _, err := b.Client.IssueTrackerApi.RepositoriesUsernameRepoSlugIssuesGet(b.Context, org, name)

		if err != nil {
			return nil, err
		}

		for _, issue := range issues.Values {
			gitIssues = append(gitIssues, BitbucketIssueToGitIssue(issue))
		}

		if issues.Next == "" {
			break
		}
	}

	return gitIssues, nil
}

func (b *BitbucketCloudProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	issues, err := b.SearchIssues(org, name, "")
	if err != nil {
		return issues, err
	}
	return FilterIssuesClosedSince(issues, t), nil
}

func (b *BitbucketCloudProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {

	issue, _, err := b.Client.IssueTrackerApi.RepositoriesUsernameRepoSlugIssuesIssueIdGet(
		b.Context,
		org,
		strconv.FormatInt(int64(number), 10),
		name,
	)

	if err != nil {
		return nil, err
	}
	return BitbucketIssueToGitIssue(issue), nil
}

func (p *BitbucketCloudProvider) IssueURL(org string, name string, number int, isPull bool) string {
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

func (b *BitbucketCloudProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {

	bIssue, _, err := b.Client.IssueTrackerApi.RepositoriesUsernameRepoSlugIssuesPost(
		b.Context,
		owner,
		repo,
		b.GitIssueToBitbucketIssue(*issue),
	)

	if err != nil {
		return nil, err
	}
	return BitbucketIssueToGitIssue(bIssue), nil
}

func (b *BitbucketCloudProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	fmt.Println("WARNING: Bitbucket Cloud doesn't support adding PR comments via the REST API")
	return nil
}

func (b *BitbucketCloudProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	fmt.Println("WARNING: Bitbucket Cloud doesn't support adding issue comments viea the REST API")
	return nil
}

func (b *BitbucketCloudProvider) HasIssues() bool {
	return true
}

func (b *BitbucketCloudProvider) IsGitHub() bool {
	return false
}

func (b *BitbucketCloudProvider) IsGitea() bool {
	return false
}

func (b *BitbucketCloudProvider) IsBitbucketCloud() bool {
	return true
}

func (b *BitbucketCloudProvider) IsBitbucketServer() bool {
	return false
}

func (b *BitbucketCloudProvider) Kind() string {
	return "bitbucketcloud"
}

// Exposed by Jenkins plugin; this one is for https://wiki.jenkins.io/display/JENKINS/BitBucket+Plugin
func (b *BitbucketCloudProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/bitbucket-scmsource-hook/notify"
}

func (b *BitbucketCloudProvider) Label() string {
	return b.Server.Label()
}

func (b *BitbucketCloudProvider) ServerURL() string {
	return b.Server.URL
}

func (p *BitbucketCloudProvider) CurrentUsername() string {
	return p.Username
}

func (p *BitbucketCloudProvider) UserAuth() auth.UserAuth {
	return p.User
}

func (p *BitbucketCloudProvider) UserInfo(username string) *GitUser {
	user, _, err := p.Client.UsersApi.UsersUsernameGet(p.Context, username)
	if err != nil {
		log.Error("Unable to fetch user info for " + username + " due to " + err.Error() + "\n")
		return nil
	}

	return &GitUser{
		Login:     username,
		Name:      user.DisplayName,
		AvatarURL: user.Links.Avatar.Href,
		URL:       user.Links.Self.Href,
	}
}

func (b *BitbucketCloudProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	fmt.Println("Bitbucket Cloud doesn't support releases")
	return nil
}

func (p *BitbucketCloudProvider) ListReleases(org string, name string) ([]*GitRelease, error) {
	answer := []*GitRelease{}
	fmt.Println("Bitbucket Cloud doesn't support releases")
	return answer, nil
}

func BitBucketCloudAccessTokenURL(url string, username string) string {
	// TODO with github we can default the scopes/flags we need on a token via adding
	// ?scopes=repo,read:user,user:email,write:repo_hook
	//
	// is there a way to do that for bitbucket?
	return util.UrlJoin(url, "/account/user", username, "/app-passwords/new")
}
