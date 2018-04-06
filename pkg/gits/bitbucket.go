package gits

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/auth"
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
		results, _, err := b.Client.TeamsApi.TeamsGet(b.Context, nil)

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
	for _, link := range bRepo.Links.Clone {
		if link.Name == "ssh" {
			sshURL = link.Href
		}
	}

	isFork := false
	if bRepo.Parent != nil {
		isFork = true
	}
	return &GitRepository{
		Name:     bRepo.Name,
		HTMLURL:  bRepo.Links.Html.Href,
		CloneURL: sshURL,
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

	if r.StatusCode == 404 {
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
	sourceFullName := fmt.Sprintf("%s/%s", b.Username, data.Repo)
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
		data.Repo,
		options,
	)

	if err != nil {
		return nil, err
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
		pr.Repo,
		prID,
	)

	if err != nil {
		return err
	}

	pr.State = &bitbucketPR.State

	if bitbucketPR.MergeCommit != nil {
		pr.MergeCommitSHA = &bitbucketPR.MergeCommit.Hash
	}
	pr.DiffURL = &bitbucketPR.Links.Diff.Href

	commits, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet(
		b.Context,
		b.Username,
		strconv.FormatInt(int64(prID), 10),
		pr.Repo,
	)

	if err != nil {
		return err
	}

	values := commits["values"].([]interface{})
	commit := values[0].(map[string]interface{})

	pr.LastCommitSha = commit["hash"].(string)

	return nil
}

func (b *BitbucketCloudProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {

	latestCommitStatus := bitbucket.Commitstatus{}

	for {
		result, _, err := b.Client.CommitstatusesApi.RepositoriesUsernameRepoSlugCommitNodeStatusesGet(
			b.Context,
			b.Username,
			pr.Repo,
			pr.LastCommitSha,
		)

		if err != nil {
			return "", err
		}

		if result.Size == 0 {
			return "", fmt.Errorf("this commit doesn't have any statuses")
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

	return latestCommitStatus.State, nil
}

func (b *BitbucketCloudProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {

	statuses := []*GitRepoStatus{}

	for {
		result, _, err := b.Client.CommitstatusesApi.RepositoriesUsernameRepoSlugCommitNodeStatusesGet(
			b.Context,
			org,
			repo,
			sha,
		)

		if err != nil {
			return nil, err
		}

		for _, status := range result.Values {

			id, err := strconv.ParseInt(status.Key, 10, 64)

			if err != nil {
				return nil, err
			}

			newStatus := &GitRepoStatus{
				ID:          id,
				URL:         status.Links.Commit.Href,
				State:       status.State,
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
		"message": message,
	}

	_, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdMergePost(
		b.Context,
		b.Username,
		strconv.FormatInt(int64(*pr.Number), 10),
		pr.Repo,
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
		},
	}
	_, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugHooksPost(
		b.Context,
		b.Username,
		data.Repo,
		options,
	)

	if err != nil {
		return err
	}
	return nil
}

func (b *BitbucketCloudProvider) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {
	return nil, nil
}

func (b *BitbucketCloudProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	return nil, nil
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
	return nil, nil
}

func (b *BitbucketCloudProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	return nil
}

func (b *BitbucketCloudProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
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

func (b *BitbucketCloudProvider) IsBitbucket() bool {
	return true
}

func (b *BitbucketCloudProvider) Kind() string {
	return "bitbucket"
}

// Exposed by Jenkins plugin; this one is for https://wiki.jenkins.io/display/JENKINS/BitBucket+Plugin
func (b *BitbucketCloudProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/bitbucket-hook/"
}

func (b *BitbucketCloudProvider) Label() string {
	return b.Server.Label()
}

func (b *BitbucketCloudProvider) ServerURL() string {
	return b.Server.URL
}

func (b *BitbucketCloudProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	return nil
}

func BitbucketAccessTokenURL(url string, username string) string {
	// TODO with github we can default the scopes/flags we need on a token via adding
	// ?scopes=repo,read:user,user:email,write:repo_hook
	//
	// is there a way to do that for bitbucket?
	return util.UrlJoin(url, "/account/user", username, "/app-passwords/new")
}
