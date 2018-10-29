package gits_test

import (
	"errors"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	mocks "github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/stretchr/testify/assert"
)

type FakeOrgLister struct {
	orgNames []string
	fail     bool
}

func (l FakeOrgLister) ListOrganisations() ([]gits.GitOrganisation, error) {
	if l.fail {
		return nil, errors.New("fail")
	}

	orgs := make([]gits.GitOrganisation, len(l.orgNames))
	for _, v := range l.orgNames {
		orgs = append(orgs, gits.GitOrganisation{Login: v})
	}
	return orgs, nil
}

func Test_getOrganizations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		testDescription string
		orgLister       gits.OrganisationLister
		userName        string
		want            []string
	}{
		{"Should return user name when ListOrganisations() fails", FakeOrgLister{fail: true}, "testuser", []string{"testuser"}},
		{"Should return user name when organization list is empty", FakeOrgLister{orgNames: []string{}}, "testuser", []string{"testuser"}},
		{"Should include user name when only 1 organization exists", FakeOrgLister{orgNames: []string{"testorg"}}, "testuser", []string{"testorg", "testuser"}},
		{"Should include user name together with all organizations when multiple exists", FakeOrgLister{orgNames: []string{"testorg", "anotherorg"}}, "testuser", []string{"anotherorg", "testorg", "testuser"}},
	}
	for _, tt := range tests {
		t.Run(tt.testDescription, func(t *testing.T) {
			result := gits.GetOrganizations(tt.orgLister, tt.userName)
			assert.Equal(t, tt.want, result)
		})
	}
}

func createAuthConfigSvc(authConfig *auth.AuthConfig) *auth.AuthConfigService {
	authConfigSvc := &auth.AuthConfigService{}
	authConfigSvc.SetConfig(authConfig)
	return authConfigSvc
}

func createAuthConfig(currentServer *auth.AuthServer, servers ...*auth.AuthServer) *auth.AuthConfig {
	servers = append(servers, currentServer)
	return &auth.AuthConfig{
		Servers:       servers,
		CurrentServer: currentServer.URL,
	}
}

func createAuthServer(url string, name string, kind string, currentUser *auth.UserAuth, users ...*auth.UserAuth) *auth.AuthServer {
	users = append(users, currentUser)
	return &auth.AuthServer{
		URL:         url,
		Name:        name,
		Kind:        kind,
		Users:       users,
		CurrentUser: currentUser.Username,
	}
}

func createGitProvider(t *testing.T, kind string, server *auth.AuthServer, user *auth.UserAuth, git gits.Gitter) gits.GitProvider {
	switch kind {
	case gits.KindGitHub:
		gitHubProvider, err := gits.NewGitHubProvider(server, user, git)
		assert.NoError(t, err, "should create GitHub provider without error")
		return gitHubProvider
	default:
		return nil
	}
}

func TestCreateGitProviderFromURL(t *testing.T) {
	t.Parallel()

	git := mocks.NewMockGitter()

	tests := []struct {
		description  string
		Name         string
		providerKind string
		hostURL      string
		git          gits.Gitter
		username     string
		apiToken     string
		batchMode    bool
		wantError    error
	}{
		{"create GitHub provider",
			"GitHub",
			gits.KindGitHub,
			"https://github.com",
			git,
			"test",
			"test",
			false,
			nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			user := &auth.UserAuth{
				Username: tc.username,
				ApiToken: tc.apiToken,
			}
			server := createAuthServer(tc.hostURL, tc.Name, tc.providerKind, user)
			authSvc := createAuthConfigSvc(createAuthConfig(server))
			result, err := gits.CreateProviderForURL(*authSvc, tc.providerKind, tc.hostURL, tc.git, tc.batchMode, nil, nil, nil)
			if tc.wantError == nil {
				assert.NoError(t, err, "should create provider without error")
			} else {
				assert.Equal(t, tc.wantError, err)
			}
			want := createGitProvider(t, tc.providerKind, server, user, tc.git)
			assertProvider(t, want, result)
		})
	}
}

func assertProvider(t *testing.T, want gits.GitProvider, result gits.GitProvider) {
	assert.Equal(t, want.Kind(), result.Kind())
	assert.Equal(t, want.ServerURL(), result.ServerURL())
	assert.Equal(t, want.UserAuth(), result.UserAuth())
}
