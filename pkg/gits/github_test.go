package gits

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GitHubProviderTestSuite struct {
	suite.Suite
	mux       *http.ServeMux
	server    *httptest.Server
	provider  GitHubProvider
	providers map[string]GitHubProvider
}

type UserProfile struct {
	name     string
	url      string
	username string
}

var profiles = []UserProfile{
	{
		url:      "https://auth.example.com",
		name:     "Test Auth Server",
		username: "test-user",
	},
	{
		url:      "https://auth.example.com",
		name:     "Test Auth Server with Underscore user",
		username: "test_user",
	},
}

var githubRouter = util.Router{
	"/repos/test-user/test-repo/issues/2/comments": util.MethodMap{
		"GET": "issue_comments.json",
	},
}

func (suite *GitHubProviderTestSuite) SetupSuite() {
	suite.mux = http.NewServeMux()

	for path, methodMap := range githubRouter {
		suite.mux.HandleFunc(path, util.GetMockAPIResponseFromFile("test_data/github", methodMap))
	}

	suite.server = httptest.NewServer(suite.mux)
	suite.Require().NotNil(suite.server)

	client := github.NewClient(nil)
	baseURL, err := url.Parse(suite.server.URL)
	suite.Require().Nil(err)

	client.BaseURL = baseURL
	suite.providers = map[string]GitHubProvider{}

	for _, profile := range profiles {
		gp, err := setupGitProvider(profile.url, profile.name, profile.username)

		suite.Require().NotNil(gp)
		suite.Require().Nil(err)

		var ok bool
		gh, ok := gp.(*GitHubProvider)

		suite.Require().NotNil(gh)
		suite.Require().True(ok)
		gh.Client = client

		suite.providers[profile.username] = *gh
	}

	suite.provider = suite.providers["test-user"]

}

func setupGitProvider(url, name, user string) (GitProvider, error) {
	as := auth.AuthServer{
		URL:         url,
		Name:        "Test Auth Server",
		Kind:        "Oauth2",
		CurrentUser: user,
	}
	ua := auth.UserAuth{
		Username: user,
		ApiToken: "0123456789abdef",
	}

	git := NewGitCLI()
	bp, err := NewGitHubProvider(&as, &ua, git)

	return bp, err
}

func TestIsOwnerGitHubUser_isOwner(t *testing.T) {
	t.Parallel()
	isOwnerGitHubUser := IsOwnerGitHubUser("owner", "owner")
	assert.True(t, isOwnerGitHubUser, "The owner should be the same as the GitHubUser")
}

func TestIsOwnerGitHubUser_isNotOwner(t *testing.T) {
	t.Parallel()
	isOwnerGitHubUser := IsOwnerGitHubUser("owner", "notowner")
	assert.False(t, isOwnerGitHubUser, "The owner must not be the same as the GitHubUser")
}

func (suite *GitHubProviderTestSuite) TestFetchComments(t *testing.T) {
	t.Parallel()

	comments, err := suite.provider.fetchComments("test-user", "test-repo", 2)
	assert.NotNil(t, comments)
	assert.Nil(t, err)
}

func (suite *GitHubProviderTestSuite) TearDownSuite() {
	suite.server.Close()
}
