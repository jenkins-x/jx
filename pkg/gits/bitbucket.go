package gits

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"golang.org/x/oauth2"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/wbrefvem/go-bitbucket"
)

// BitbucketProvider implements GitProvider interface for bitbucket.org
type BitbucketProvider struct {
	Client   *bitbucket.APIClient
	Username string
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
}

func NewBitbucketProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	ctx := context.Background()

	provider := BitbucketProvider{
		Server:   *server,
		User:     *user,
		Username: user.Username,
		Context:  ctx,
	}

	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: user.ApiToken},
	)
	tokenContext := oauth2.NewClient(ctx, tokenSource)

	cfg := bitbucket.NewConfiguration()
	cfg.HTTPClient = tokenContext

	provider.Client = bitbucket.NewAPIClient(cfg)

	return &provider, nil
}

func (b *BitbucketProvider) ListOrganisations() ([]GitOrganisation, error) {

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

func repoFromRepo(bRepo bitbucket.Repository) *GitRepository {
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

func (b *BitbucketProvider) ListRepositories(org string) ([]*GitRepository, error) {

	repos := []*GitRepository{}

	for {
		results, _, err := b.Client.RepositoriesApi.RepositoriesUsernameGet(b.Context, org, nil)

		if err != nil {
			return nil, err
		}

		for _, repo := range results.Values {
			repos = append(repos, repoFromRepo(repo))
		}

		if results.Next == "" {
			break
		}
	}

	return repos, nil
}

func (b *BitbucketProvider) CreateRepository(
	org string,
	name string,
	private bool,
) (*GitRepository, error) {

	var options map[string]interface{}
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

	return repoFromRepo(result), nil
}

func (b *BitbucketProvider) GetRepository(
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

	return repoFromRepo(repo), nil
}

func (b *BitbucketProvider) DeleteRepository(org string, name string) error {

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

func (b *BitbucketProvider) ForkRepository(
	originalOrg string,
	name string,
	destinationOrg string,
) (*GitRepository, error) {

	repo, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugForksPost(
		b.Context,
		b.Username,
		name,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return repoFromRepo(repo), nil
}

func (b *BitbucketProvider) RenameRepository(
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

	return repoFromRepo(repo), nil
}

func (b *BitbucketProvider) ValidateRepositoryName(org string, name string) error {

	_, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugGet(
		b.Context,
		b.Username,
		name,
	)

	if err == nil {
		return fmt.Errorf("repository %s/%s already exists", b.Username, name)
	}

	return err
}

func (b *BitbucketProvider) CreatePullRequest(
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

func (b *BitbucketProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {

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

func (b *BitbucketProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	return "", nil
}

func (b *BitbucketProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	return nil, nil
}

func (b *BitbucketProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	return nil
}

func (b *BitbucketProvider) CreateWebHook(data *GitWebHookArguments) error {
	return nil
}

func (b *BitbucketProvider) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {
	return nil, nil
}

func (b *BitbucketProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	return nil, nil
}

func (p *BitbucketProvider) IssueURL(org string, name string, number int, isPull bool) string {
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

func (b *BitbucketProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	return nil, nil
}

func (b *BitbucketProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	return nil
}

func (b *BitbucketProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	return nil
}

func (b *BitbucketProvider) HasIssues() bool {
	return true
}

func (b *BitbucketProvider) IsGitHub() bool {
	return false
}

func (b *BitbucketProvider) IsGitea() bool {
	return false
}

func (b *BitbucketProvider) IsBitbucket() bool {
	return true
}

func (b *BitbucketProvider) Kind() string {
	return "bitbucket"
}

func (b *BitbucketProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return ""
}

func (b *BitbucketProvider) Label() string {
	return ""
}

func (b *BitbucketProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	return nil
}
