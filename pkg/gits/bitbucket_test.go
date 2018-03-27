package gits

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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
		options map[string]interface{}
	) (bitbucket.PaginatedTeams, *http.Response, error) {

	return nil, nil, nil
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

	bitbucketProvider, err := bp.(*BitbucketProvider)

	assert.Nil(err)
}
