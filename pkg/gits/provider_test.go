package gits_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	mocks "github.com/jenkins-x/jx/pkg/gits/mocks"
	utiltests "github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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

func createAuthConfigSvc(authConfig *auth.AuthConfig, fileName string) *auth.AuthConfigService {
	authConfigSvc := &auth.AuthConfigService{
		FileName: fileName,
	}
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

func setUserAuthInEnv(kind string, username string, apiToken string) error {
	prefix := strings.ToUpper(kind)
	err := os.Setenv(prefix+"_USERNAME", username)
	if err != nil {
		return err
	}
	return os.Setenv(prefix+"_API_TOKEN", apiToken)
}

func unsetUserAuthInEnv(kind string) error {
	prefix := strings.ToUpper(kind)
	err := os.Unsetenv(prefix + "_USERNAME")
	if err != nil {
		return err
	}
	return os.Unsetenv(prefix + "_API_TOKEN")
}

func TestCreateGitProviderFromURL(t *testing.T) {
	t.Parallel()

	git := mocks.NewMockGitter()

	tests := []struct {
		description  string
		setup        func(t *testing.T) (*expect.Console, *terminal.Stdio, chan struct{})
		cleanup      func(t *testing.T, c *expect.Console, donech chan struct{})
		Name         string
		providerKind string
		hostURL      string
		git          gits.Gitter
		numUsers     int
		currUser     int
		username     string
		apiToken     string
		batchMode    bool
		wantError    bool
	}{
		{"create GitHub provider for one user",
			nil,
			nil,
			"GitHub",
			gits.KindGitHub,
			"https://github.com",
			git,
			1,
			0,
			"test",
			"test",
			false,
			false,
		},
		{"create GitHub provider for multiple users",
			nil,
			nil,
			"GitHub",
			gits.KindGitHub,
			"https://github.com",
			git,
			2,
			1,
			"test",
			"test",
			false,
			false,
		},
		{"create GitHub provider for user from environment",
			func(t *testing.T) (*expect.Console, *terminal.Stdio, chan struct{}) {
				err := setUserAuthInEnv(gits.KindGitHub, "test", "test")
				assert.NoError(t, err, "should configure the user auth in environment")
				c, _, term := utiltests.NewTerminal(t)
				donech := make(chan struct{})
				go func() {
					defer close(donech)
				}()
				return c, term, donech
			},
			func(t *testing.T, c *expect.Console, donech chan struct{}) {
				err := unsetUserAuthInEnv(gits.KindGitHub)
				assert.NoError(t, err, "should reset the user auth in environment")
				err = c.Tty().Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"GitHub",
			gits.KindGitHub,
			"https://github.com",
			git,
			0,
			0,
			"test",
			"test",
			false,
			false,
		},
		{"create GitHub provider in barch mode ",
			nil,
			nil,
			"GitHub",
			gits.KindGitHub,
			"https://github.com",
			git,
			0,
			0,
			"",
			"",
			true,
			true,
		},
		{"create GitHub provider for in interactive mode",
			func(t *testing.T) (*expect.Console, *terminal.Stdio, chan struct{}) {
				c, _, term := utiltests.NewTerminal(t)
				assert.NotNil(t, c, "console should not be nil")
				assert.NotNil(t, term, "term should not be nil")
				donech := make(chan struct{})
				go func() {
					defer close(donech)
					_, err := c.ExpectString("github.com user name:")
					assert.NoError(t, err, "expect user name")
					_, err = c.SendLine("test")
					assert.NoError(t, err, "send user name")
					_, err = c.ExpectString("API Token:")
					assert.NoError(t, err, "expect API token")
					_, err = c.SendLine("test")
					assert.NoError(t, err, "send API token")
					_, err = c.ExpectEOF()
					assert.NoError(t, err, "expect EOF")
				}()
				return c, term, donech
			},
			func(t *testing.T, c *expect.Console, donech chan struct{}) {
				err := c.Tty().Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"GitHub",
			gits.KindGitHub,
			"https://github.com",
			git,
			0,
			0,
			"test",
			"test",
			false,
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			var console *expect.Console
			var term *terminal.Stdio
			var donech chan struct{}
			if tc.setup != nil {
				console, term, donech = tc.setup(t)
			}
			users := []*auth.UserAuth{}
			var currUser *auth.UserAuth
			var server *auth.AuthServer
			var authSvc *auth.AuthConfigService

			configFile, err := ioutil.TempFile("", "test-config")
			defer os.Remove(configFile.Name())
			if tc.numUsers > 0 {
				for u := 1; u <= tc.numUsers; u++ {
					user := &auth.UserAuth{
						Username: fmt.Sprintf("%s-%d", tc.username, u),
						ApiToken: fmt.Sprintf("%s-%d", tc.apiToken, u),
					}
					users = append(users, user)
				}
				assert.True(t, len(users) > tc.currUser, "current user index should be smaller than number of users")
				currUser = users[tc.currUser]
				if len(users) > 1 {
					users = append(users[:tc.currUser], users[tc.currUser+1:]...)
				} else {
					users = []*auth.UserAuth{}
				}
				server = createAuthServer(tc.hostURL, tc.Name, tc.providerKind, currUser, users...)
				authSvc = createAuthConfigSvc(createAuthConfig(server), configFile.Name())
			} else {
				currUser = &auth.UserAuth{
					Username: tc.username,
					ApiToken: tc.apiToken,
				}
				server = createAuthServer(tc.hostURL, tc.Name, tc.providerKind, currUser, users...)
				authSvc = &auth.AuthConfigService{
					FileName: configFile.Name(),
				}
			}

			var result gits.GitProvider
			if term != nil {
				result, err = gits.CreateProviderForURL(*authSvc, tc.providerKind, tc.hostURL, tc.git, tc.batchMode, term.In, term.Out, term.Err)
			} else {
				result, err = gits.CreateProviderForURL(*authSvc, tc.providerKind, tc.hostURL, tc.git, tc.batchMode, nil, nil, nil)
			}
			if tc.wantError {
				assert.Error(t, err, "should fail to create provider")
				assert.Nil(t, result, "created provider should be nil")
			} else {
				assert.NoError(t, err, "should create provider without error")
				assert.NotNil(t, result, "created provider should not be nil")
				want := createGitProvider(t, tc.providerKind, server, currUser, tc.git)
				assert.NotNil(t, want, "expected provider should not be nil")
				assertProvider(t, want, result)
			}
			if tc.cleanup != nil {
				tc.cleanup(t, console, donech)
			}
		})
	}
}

func assertProvider(t *testing.T, want gits.GitProvider, result gits.GitProvider) {
	assert.Equal(t, want.Kind(), result.Kind())
	assert.Equal(t, want.ServerURL(), result.ServerURL())
	assert.Equal(t, want.UserAuth(), result.UserAuth())
}
