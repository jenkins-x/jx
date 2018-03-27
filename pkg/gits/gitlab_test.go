package gits

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/suite"
	"github.com/xanzy/go-gitlab"
	"io/ioutil"
	"fmt"
)

const (
	userName = "testperson"
	orgName  = "testorg"
)

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
		Username: userName,
		ApiToken: "test",
	}
	// Gitlab provider that we want to test
	provider, _ := withGitlabClient(new(auth.AuthServer), userAuth, client)

	return mux, server, provider.(*GitlabProvider)
}

func configureGitlabMock(suite *GitlabProviderSuite, mux *http.ServeMux) {
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test-fixtures/groups.json")

		suite.Require().Nil(err)
		w.Write(src)
	})

	mux.HandleFunc(fmt.Sprintf("/groups/%s/projects", orgName), func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test-fixtures/group-projects.json")

		suite.Require().Nil(err)
		w.Write(src)
	})

	mux.HandleFunc(fmt.Sprintf("/users/%s/projects", userName), func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test-fixtures/user-projects.json")

		suite.Require().Nil(err)
		w.Write(src)
	})
}

func (suite *GitlabProviderSuite) TestListOrganizations() {
	orgs, err := suite.provider.ListOrganisations()

	suite.Require().Nil(err)
	suite.Require().Len(orgs, 1)
	suite.Require().Equal(orgName, orgs[0].Login)
}

func (suite *GitlabProviderSuite) TestListRepositories() {
	require := suite.Require()
	scenarios := []struct {
		testDescription  string
		org              string
		expectedRepoName string
		expectedSshUrl   string
		expectedHtmlUrl  string
	} {
		{"List repositories for organization", orgName, "orgproject", "git@gitlab.com:testorg/orgproject.git", "https://gitlab.com/testorg/orgproject"},
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

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestGitlabProviderSuite(t *testing.T) {
	suite.Run(t, new(GitlabProviderSuite))
}
