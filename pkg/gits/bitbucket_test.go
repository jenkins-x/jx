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

func handleNotFound(response http.ResponseWriter, err error) {
	response.WriteHeader(http.StatusNotFound)
	response.Write([]byte(err.Error()))
}

func handleOk(response http.ResponseWriter, body []byte) {
	response.WriteHeader(http.StatusOK)
	response.Write(body)
}

// Are you a mod or a rocker? I'm a
type mocker func(http.ResponseWriter, *http.Request)

// "get" as in getter not HTTP GET; supports all methods since this is a mock.
func getMockAPIResponseFromFile(dataDir string, fileName string) mocker {

	return func(response http.ResponseWriter, request *http.Request) {
		obj, err := util.LoadBytes(dataDir, fileName)

		if err != nil {
			handleNotFound(response, err)
		}

		handleOk(response, obj)
	}
}

func (suite *BitbucketProviderTestSuite) SetupSuite() {
	suite.mux = http.NewServeMux()

	suite.mux.HandleFunc(
		"/repositories/test-user",
		getMockAPIResponseFromFile("test_data/bitbucket", "repos.json"),
	)
	suite.mux.HandleFunc(
		"/repositories/test-user/python-jolokia",
		getMockAPIResponseFromFile("test_data/bitbucket", "repos.test-repo.json"),
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

	repos, _, err := suite.provider.Client.RepositoriesApi.RepositoriesUsernameGet(
		suite.provider.Context,
		suite.provider.Username,
		nil,
	)

	suite.Require().Nil(err)
	suite.Require().NotNil(repos)
	suite.Require().NotNil(repos.Values)

	suite.Require().Equal(repos.Size, int32(2))

	for _, repo := range repos.Values {
		suite.Require().NotNil(repo)
		suite.Require().Equal(repo.Owner.Username, "test-user")
	}
}

func (suite *BitbucketProviderTestSuite) TestGetRepository() {

	repo, _, err := suite.provider.Client.RepositoriesApi.RepositoriesUsernameRepoSlugGet(
		suite.provider.Context,
		"test-user",
		"python-jolokia",
	)

	suite.Require().NotNil(repo)
	suite.Require().Nil(err)

	suite.Require().Equal(repo.Owner.Username, "test-user")
}

func TestBitbucketProviderTestSuite(t *testing.T) {
	suite.Run(t, new(BitbucketProviderTestSuite))
}

func (suite *BitbucketProviderTestSuite) TearDownSuite() {
	suite.server.Close()
}
