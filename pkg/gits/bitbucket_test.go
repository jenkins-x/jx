package gits

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	username = "test-user"
	orgName  = "test-org"
)

type BitbucketProviderSuite struct {
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

func (suite *BitbucketProviderSuite) SetupSuite() {
	mux := http.NewServeMux()

	mux.HandleFunc(fmt.Sprintf("/respositories/%s", username), getMockAPIResponseFromFile("test_data", "teams.json"))
}

func TestListOrganisations(t *testing.T) {

	as := auth.AuthServer{
		URL:         "https://auth.example.com",
		Name:        "Test Auth Server",
		Kind:        "Oauth2",
		CurrentUser: "wbrefvem",
	}
	ua := auth.UserAuth{
		Username: "wbrefvem",
		ApiToken: "0123456789abdef",
	}

	bp, err := NewBitbucketProvider(&as, &ua)

	assert.Nil(t, err)
	assert.NotNil(t, bp)

	bitbucketProvider, ok := bp.(*BitbucketProvider)

	assert.True(t, ok)
	assert.NotNil(t, bitbucketProvider)
}
