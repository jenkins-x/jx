package gits

import (
	"context"
	"net/http"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	bitbucket "github.com/wbrefvem/go-bitbucket"
)

type MockBitbucketAPIClient struct {
	mock.Mock
}

type MockTeamsApi struct {
	mock.Mock
}

type MockPullrequestsApi struct {
	mock.Mock
}

type MockRepositoriesApi struct {
	mock.Mock
}

type MockCommitsApi struct {
	mock.Mock
}

func (mbAPIc *MockBitbucketAPIClient) MockTeamsGet200OK(
	ctx context.Context,
	options map[string]interface{},
) (bitbucket.PaginatedTeams, *http.Response, error) {
	var teams bitbucket.PaginatedTeams
	return teams, nil, nil
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
