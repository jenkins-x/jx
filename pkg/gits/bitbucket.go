package gits

import (
	"context"
	"fmt"
	"net/http"
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

func handleErrorStatusCode(response *http.Response) error {
	return fmt.Errorf("Bitbucket API returned status %s with the following explanation: %s", response.Status, response.Body)
}

func (b *BitbucketProvider) ListRepositories(org string) ([]*GitRepository, error) {

	repos := []*GitRepository{}

	for {
		results, response, err := b.Client.RepositoriesApi.RepositoriesGet(b.Context, nil)

		if err != nil {
			return nil, err
		}

		// https://i.giphy.com/media/12NUbkX6p4xOO4/giphy.webp
		if response.StatusCode >= 400 {
			return nil, handleErrorStatusCode(response)
		}

		for _, repo := range results.Values {
			repos = append(repos, repoFromRepo(repo))
		}

		if results.Next == "" {
			break
		}
	}

	return nil, nil
}

func (b *BitbucketProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {

	var options map[string]interface{}
	options["body"] = bitbucket.Repository{
		IsPrivate: private,
	}

	result, response, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugPost(b.Context, b.Username, name, options)

	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 400 {
		return nil, handleErrorStatusCode(response)
	}

	return repoFromRepo(result), nil
}

func (b *BitbucketProvider) GetRepository(org string, name string) (*GitRepository, error) {

	repo, response, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugGet(b.Context, org, name)

	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 400 {
		return nil, handleErrorStatusCode(response)
	}

	return repoFromRepo(repo), nil
}

func (b *BitbucketProvider) DeleteRepository(org string, name string) error {

	response, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugDelete(b.Context, b.Username, name, nil)

	if err != nil {
		return err
	}

	if response.StatusCode >= 400 {
		return handleErrorStatusCode(response)
	}

	return nil
}

func (b *BitbucketProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {

	repo, response, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugForksPost(b.Context, b.Username, name, nil)

	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 400 {
		return nil, handleErrorStatusCode(response)
	}

	return repoFromRepo(repo), err
}

func (b *BitbucketProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {

	var options map[string]interface{}

	options["name"] = newName

	repo, response, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugPut(b.Context, b.Username, name, options)

	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 400 {
		return nil, handleErrorStatusCode(response)
	}

	return repoFromRepo(repo), nil
}

func (b *BitbucketProvider) ValidateRepositoryName(org string, name string) error {

	_, response, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugGet(b.Context, b.Username, name)

	if err == nil && response.StatusCode == http.StatusOK {
		return fmt.Errorf("Repository %s/%s already exists!", b.Username, name)
	} else if err == nil && response.StatusCode >= 400 {
		return handleErrorStatusCode(response)
	}

	if response.StatusCode == 404 {
		return nil
	}
	return err
}

func (b *BitbucketProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	return nil, nil
}

func (b *BitbucketProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
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
