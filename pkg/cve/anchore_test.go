package cve

import (
	"testing"

	"os"

	"net/http"

	"net/http/httptest"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/suite"
)

type AnchoreProviderTestSuite struct {
	suite.Suite
	mux      *http.ServeMux
	server   *httptest.Server
	provider *AnchoreProvider
}

var router = util.Router{
	"/images/by_id/07b67913cd8c1ffc961c402b58c4e539ee6aaeae0b08969fc653267f4b975503/vuln/os": util.MethodMap{
		"GET": "vulnerabilities.json",
	},
}

func TestAnchoreProviderTestSuite(t *testing.T) {
	suite.Run(t, new(AnchoreProviderTestSuite))
}

func (suite *AnchoreProviderTestSuite) SetupSuite() {
	suite.mux = http.NewServeMux()

	for path, methodMap := range router {
		suite.mux.HandleFunc(path, util.GetMockAPIResponseFromFile("test_data/anchore", methodMap))
	}

	suite.server = httptest.NewServer(suite.mux)
	suite.Require().NotNil(suite.server)

	as := auth.AuthServer{
		URL:         suite.server.URL,
		Name:        "Test Anchore Server",
		Kind:        "anchore-anchore-engine-core",
		CurrentUser: "admin",
	}
	ua := auth.UserAuth{
		Username: "admin",
		ApiToken: "admin",
	}

	a, err := NewAnchoreProvider(&as, &ua)
	suite.Require().NotNil(a)
	suite.Require().Nil(err)

	var ok bool
	suite.provider, ok = a.(*AnchoreProvider)
	suite.Require().True(ok)
	suite.Require().NotNil(suite.provider)

}

func (suite *AnchoreProviderTestSuite) TestGetImageVulnerabilityTable() {

	vTable := table.CreateTable(os.Stdout)

	image := "07b67913cd8c1ffc961c402b58c4e539ee6aaeae0b08969fc653267f4b975503"
	err := suite.provider.GetImageVulnerabilityTable(&vTable, image)
	suite.Require().NoError(err)

	vTable.Render()

}
