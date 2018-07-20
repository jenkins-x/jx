package gits

import (
	"testing"

	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/suite"
	"github.com/xanzy/go-gitlab"
)

const (
	gitlabUserName = "testperson"
	gitlabOrgName  = "testorg"
)

var gitlabRouter = util.Router{
	"/api/v4/projects/testperson%2Ftest-project": util.MethodMap{
		"GET": "project.json",
	},
}

type GitlabProviderSuite struct {
	suite.Suite
	mux      *http.ServeMux
	server   *httptest.Server
	provider *GitlabProvider
}

func (suite *GitlabProviderSuite) SetupSuite() {
	mux, server, provider := setup(suite)
	suite.mux = mux
	suite.server = server
	suite.provider = provider
}

func (suite *GitlabProviderSuite) TearDownSuite() {
	suite.server.Close()
}

// setup sets up a test HTTP server along with a gitlab.Client that is
// configured to talk to that test server.  Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setup(suite *GitlabProviderSuite) (*http.ServeMux, *httptest.Server, *GitlabProvider) {
	// mux is the HTTP request multiplexer used with the test server.
	mux := http.NewServeMux()
	configureGitlabMock(suite, mux)

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(mux)

	// Gitlab client
	client := gitlab.NewClient(nil, "")
	client.SetBaseURL(server.URL)

	userAuth := &auth.UserAuth{
		Username: gitlabUserName,
		ApiToken: "test",
	}
	// Gitlab provider that we want to test
	git := NewGitCLI()
	provider, _ := withGitlabClient(new(auth.AuthServer), userAuth, client, git)

	return mux, server, provider.(*GitlabProvider)
}

func configureGitlabMock(suite *GitlabProviderSuite, mux *http.ServeMux) {
	mux.HandleFunc("/api/v4/groups", func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test_data/gitlab/groups.json")

		suite.Require().Nil(err)
		w.Write(src)
	})

	mux.HandleFunc(fmt.Sprintf("/api/v4/groups/%s/projects", gitlabOrgName), func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test_data/gitlab/group-projects.json")

		suite.Require().Nil(err)
		w.Write(src)
	})

	mux.HandleFunc(fmt.Sprintf("/api/v4/users/%s/projects", gitlabUserName), func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test_data/gitlab/user-projects.json")

		suite.Require().Nil(err)
		w.Write(src)
	})

	for path, methodMap := range gitlabRouter {
		mux.HandleFunc(path, util.GetMockAPIResponseFromFile("test_data/gitlab", methodMap))
	}

	suite.T().Logf("Escape encoded: %v", url.QueryEscape("testperson%2Ftest-project"))
	suite.T().Logf("Escape unencoded: %v", url.QueryEscape("testperson/test-project"))

}

func (suite *GitlabProviderSuite) TestListOrganizations() {
	orgs, err := suite.provider.ListOrganisations()

	suite.Require().Nil(err)
	suite.Require().Len(orgs, 1)
	suite.Require().Equal(gitlabOrgName, orgs[0].Login)
}

func (suite *GitlabProviderSuite) TestListRepositories() {
	require := suite.Require()
	scenarios := []struct {
		testDescription  string
		org              string
		expectedRepoName string
		expectedSshUrl   string
		expectedHtmlUrl  string
	}{
		{"List repositories for organization", gitlabOrgName, "orgproject", "git@gitlab.com:testorg/orgproject.git", "https://gitlab.com/testorg/orgproject"},
		{"List repositories without organization", "", "userproject", "git@gitlab.com:testperson/userproject.git", "https://gitlab.com/testperson/userproject"},
	}

	for _, s := range scenarios {
		repositories, err := suite.provider.ListRepositories(s.org)
		require.Nil(err)
		require.Len(repositories, 1)
		require.Equal(s.expectedRepoName, repositories[0].Name)
		require.Equal(s.expectedSshUrl, repositories[0].SSHURL)
		require.Equal(s.expectedSshUrl, repositories[0].CloneURL)
		require.Equal(s.expectedHtmlUrl, repositories[0].HTMLURL)
	}
}

func (suite *GitlabProviderSuite) TestGetRepository() {
	repo, err := suite.provider.GetRepository("testperson", "test-project")

	suite.Require().Nil(err)
	suite.Require().NotNil(repo)

	suite.Require().Equal(repo.Name, "test-project")
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestGitlabProviderSuite(t *testing.T) {
	suite.Run(t, new(GitlabProviderSuite))
}
