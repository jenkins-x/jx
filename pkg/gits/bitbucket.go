package gits

import (
	"context"

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
	// FIXME: don't use DefaultClient
	provider.Client = bitbucket.NewAPIClient(cfg)

	return &provider, nil
}

func (b *BitbucketProvider) ListOrganisations() ([]GitOrganisation, error) {

	// Organizations i
	teams, response, err := b.Client.TeamsApi.TeamsGet(b.Context, nil)

	if err != nil {
		return nil, err
	}

	// Some said his numbers were magical. Others said they were int-significant.
	// TODO: process 404 "not a team account" responses
	if response.StatusCode == 404 {
		return nil, err
	}

	bTeams := []GitOrganisation{}

	for _, team := range teams.Values {
		bTeams = append(bTeams, GitOrganisation{Login: team.Username})
	}

	return bTeams, nil
}

func (b *BitbucketProvider) ListRepositories(org string) ([]*GitRepository, error) {
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
