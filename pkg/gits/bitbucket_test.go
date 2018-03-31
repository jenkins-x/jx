package gits

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/suite"
	bitbucket "github.com/wbrefvem/go-bitbucket"
)

const (
	username = "test-user"
	orgName  = "test-org"
)

type BitbucketProviderTestSuite struct {
	suite.Suite
	mux      *http.ServeMux
	server   *httptest.Server
	provider *BitbucketProvider
}

func handleErr(response http.ResponseWriter, err error) {
	response.WriteHeader(http.StatusInternalServerError)
	response.Write([]byte(err.Error()))
}

func handleOk(response http.ResponseWriter, body []byte) {
	response.WriteHeader(http.StatusOK)
	response.Write(body)
}

var router = map[string]interface{}{
	"/repositories/test-user": map[string]interface{}{
		"GET": "repos.json",
	},
	"/repositories/test-user/test-repo": map[string]interface{}{
		"GET":    "repos.test-repo.json",
		"DELETE": "repos.test-repo.nil.json",
	},
	"/repositories/test-user/test-repo/forks": map[string]interface{}{
		"POST": "repos.test-fork.json",
	},
}

// Are you a mod or a rocker? I'm a
type mocker func(http.ResponseWriter, *http.Request)

// TODO: Find a DRY abstraction for mapping (url, method) -> file
// e.g., ("/foo", "PUT") -> updated-foo.json
func getMockAPIResponseFromFile(dataDir string) mocker {

	return func(response http.ResponseWriter, request *http.Request) {
		route := router[request.URL.Path].(map[string]interface{})
		fileName := route[request.Method].(string)

		obj, err := util.LoadBytes(dataDir, fileName)

		if err != nil {
			handleErr(response, err)
		}

		handleOk(response, obj)
	}
}

func (suite *BitbucketProviderTestSuite) SetupSuite() {
	suite.mux = http.NewServeMux()

	suite.mux.HandleFunc(
		"/repositories/test-user",
		getMockAPIResponseFromFile("test_data/bitbucket"),
	)
	suite.mux.HandleFunc(
		"/repositories/test-user/test-repo",
		getMockAPIResponseFromFile("test_data/bitbucket"),
	)
	suite.mux.HandleFunc(
		"/repositories/test-user/test-repo/forks",
		getMockAPIResponseFromFile("test_data/bitbucket"),
	)

	as := auth.AuthServer{
		URL:         "https://auth.example.com",
		Name:        "Test Auth Server",
		Kind:        "Oauth2",
		CurrentUser: "test-user",
	}
	ua := auth.UserAuth{
		Username: "test-user",
		ApiToken: "0123456789abdef",
	}

	bp, err := NewBitbucketProvider(&as, &ua)

	suite.Require().NotNil(bp)
	suite.Require().Nil(err)

	var ok bool
	suite.provider, ok = bp.(*BitbucketProvider)
	suite.Require().True(ok)
	suite.Require().NotNil(suite.provider)

	suite.server = httptest.NewServer(suite.mux)
	suite.Require().NotNil(suite.server)

	cfg := bitbucket.NewConfiguration()
	cfg.BasePath = suite.server.URL

	suite.provider.Client = bitbucket.NewAPIClient(cfg)
}

func (suite *BitbucketProviderTestSuite) TestListRepositories() {

	repos, err := suite.provider.ListRepositories("test-user")

	suite.Require().Nil(err)
	suite.Require().NotNil(repos)

	suite.Require().Equal(len(repos), 2)

	for _, repo := range repos {
		suite.Require().NotNil(repo)
	}
}

func (suite *BitbucketProviderTestSuite) TestGetRepository() {

	repo, err := suite.provider.GetRepository(
		suite.provider.Username,
		"test-repo",
	)

	suite.Require().NotNil(repo)
	suite.Require().Nil(err)

	suite.Require().Equal(repo.Name, "test-repo")
}

func (suite *BitbucketProviderTestSuite) TestDeleteRepository() {

	err := suite.provider.DeleteRepository(
		suite.provider.Username,
		"test-repo",
	)

	suite.Require().Nil(err)
}

func (suite *BitbucketProviderTestSuite) TestForkRepository() {

	fork, err := suite.provider.ForkRepository(
		suite.provider.Username,
		"test-repo",
		"",
	)

	suite.Require().NotNil(fork)
	suite.Require().Nil(err)
}

func (suite *BitbucketProviderTestSuite) TestValidateRepositoryName() {

	err := suite.provider.ValidateRepositoryName(suite.provider.Username, "test-repo")

	suite.Require().NotNil(err)

	err = suite.provider.ValidateRepositoryName(suite.provider.Username, "foo-repo")

	suite.Require().NotNil(err)
}

func TestBitbucketProviderTestSuite(t *testing.T) {
	suite.Run(t, new(BitbucketProviderTestSuite))
}

func (suite *BitbucketProviderTestSuite) TearDownSuite() {
	suite.server.Close()
}
