package gits

import (
	"context"
	"net/http"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/wbrefvem/go-bitbucket"
)

// BitbucketProvider mplements GitProvider interface for bitbucket.org
type BitbucketProvider struct {
	Client   *bitbucket.APIClient
	Username string
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
}

// NewBitbucketProvider is a constructor for type BitbucketProvider
func NewBitbucketProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	ctx := context.Background()

	basicAuthContext := context.WithValue(
		ctx,
		bitbucket.ContextBasicAuth,
		bitbucket.BasicAuth{
			UserName: user.Username,
			// App Password, equivalent to GH personal access token
			Password: user.ApiToken,
		},
	)

	provider := BitbucketProvider{
		Server:   *server,
		User:     *user,
		Username: user.Username,
		Context:  basicAuthContext,
	}

	cfg := bitbucket.NewConfiguration()
	cfg.HTTPClient = &http.Client{
		Timeout: 30 * time.Second,
	}

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

func (b *BitbucketProvider) ListRepositories(org string) ([]*GitRepository, error) {

	repos := []*GitRepository{}

	for {
		results, _, err := b.Client.RepositoriesApi.RepositoriesGet(b.Context, nil)

		if err != nil {
			return nil, err
		}

		for _, repo := range results.Values {

			var sshURL string
			for _, link := range repo.Links.Clone {
				if link.Name == "ssh" {
					sshURL = link.Href
				}
			}

			isFork := false
			if repo.Parent != nil {
				isFork = true
			}

			repos = append(
				repos,
				&GitRepository{
					Name:     repo.Name,
					HTMLURL:  repo.Links.Html.Href,
					CloneURL: sshURL,
					SSHURL:   sshURL,
					Language: repo.Language,
					Fork:     isFork,
				},
			)
		}

		if results.Next == "" {
			break
		}
	}

	return nil, nil
}

func (b *BitbucketProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	return nil, nil
}

func (b *BitbucketProvider) GetRepository(org string, name string) (*GitRepository, error) {
	return nil, nil
}

func (b *BitbucketProvider) DeleteRepository(org string, name string) error {
	return nil
}

func (b *BitbucketProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	return nil, nil
}

func (b *BitbucketProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	return nil, nil
}

func (b *BitbucketProvider) ValidateRepositoryName(org string, name string) error {
	return nil
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

func (b *BitbucketProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	return nil, nil
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
