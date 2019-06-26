package gits_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	mocks "github.com/jenkins-x/jx/pkg/gits/mocks"
	utiltests "github.com/jenkins-x/jx/pkg/tests"
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

func createAuthServer(url string, name string, kind string, currentUser auth.User, users ...auth.User) auth.Server {
	users = append(users, currentUser)
	return auth.Server{
		URL:         url,
		Name:        name,
		Kind:        kind,
		Users:       users,
		CurrentUser: currentUser.Username,
	}
}

func createGitProvider(t *testing.T, kind string, server auth.Server, git gits.Gitter) gits.GitProvider {
	switch kind {
	case gits.KindGitHub:
		gitHubProvider, err := gits.NewGitHubProvider(server, git)
		assert.NoError(t, err, "should create GitHub provider without error")
		return gitHubProvider
	case gits.KindGitlab:
		gitlabProvider, err := gits.NewGitlabProvider(server, git)
		assert.NoError(t, err, "should create Gitlab provider without error")
		return gitlabProvider
	case gits.KindGitea:
		giteaProvider, err := gits.NewGiteaProvider(server, git)
		assert.NoError(t, err, "should create Gitea provider without error")
		return giteaProvider
	case gits.KindBitBucketServer:
		bitbucketServerProvider, err := gits.NewBitbucketServerProvider(server, git)
		assert.NoError(t, err, "should create Bitbucket server  provider without error")
		return bitbucketServerProvider
	case gits.KindBitBucketCloud:
		bitbucketCloudProvider, err := gits.NewBitbucketCloudProvider(server, git)
		assert.NoError(t, err, "should create Bitbucket cloud  provider without error")
		return bitbucketCloudProvider
	default:
		return nil
	}
}

func TestCreateGitProviderFromURL(t *testing.T) {
	t.Parallel()
	utiltests.SkipForWindows(t, "go-expect does not work on Windows")

	git := mocks.NewMockGitter()

	tests := []struct {
		description  string
		Name         string
		providerKind string
		hostURL      string
		git          gits.Gitter
		numUsers     int
		currUser     int
		username     string
		apiToken     string
		wantError    bool
	}{
		{"create GitHub provider for one user",
			"GitHub",
			gits.KindGitHub,
			"https://github.com",
			git,
			1,
			0,
			"test",
			"test",
			false,
		},
		{"create GitHub provider for multiple users",
			"GitHub",
			gits.KindGitHub,
			"https://github.com",
			git,
			2,
			1,
			"test",
			"test",
			false,
		},
		{"create Gitlab provider for one user",
			"Gitlab",
			gits.KindGitlab,
			"https://gitlab.com",
			git,
			1,
			0,
			"test",
			"test",
			false,
		},
		{"create Gitlab provider for multiple users",
			"Gitlab",
			gits.KindGitHub,
			"https://gitlab.com",
			git,
			2,
			1,
			"test",
			"test",
			false,
		},
		{"create Gitea provider for one user",
			"Gitea",
			gits.KindGitea,
			"https://gitea.com",
			git,
			1,
			0,
			"test",
			"test",
			false,
		},
		{"create Gitea provider for multiple users",
			"Gitea",
			gits.KindGitea,
			"https://gitea.com",
			git,
			2,
			1,
			"test",
			"test",
			false,
		},
		{"create BitbucketServer provider for one user",
			"BitbucketServer",
			gits.KindBitBucketServer,
			"https://bitbucket-server.com",
			git,
			1,
			0,
			"test",
			"test",
			false,
		},
		{"create BitbucketServer provider for multiple users",
			"BitbucketServer",
			gits.KindBitBucketServer,
			"https://bitbucket-server.com",
			git,
			2,
			1,
			"test",
			"test",
			false,
		},
		{"create BitbucketCloud provider for one user",
			"BitbucketCloud",
			gits.KindBitBucketCloud,
			"https://bitbucket.org",
			git,
			1,
			0,
			"test",
			"test",
			false,
		},
		{"create BitbucketCloud provider for multiple users",
			"BitbucketCloud",
			gits.KindBitBucketCloud,
			"https://bitbucket.org",
			git,
			2,
			1,
			"test",
			"test",
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			var users []auth.User
			var currUser auth.User
			var server auth.Server
			configFile, err := ioutil.TempFile("", "test-config")
			assert.NoError(t, err, "should create temp file")
			defer os.Remove(configFile.Name())
			if tc.numUsers > 0 {
				for u := 1; u <= tc.numUsers; u++ {
					user := auth.User{
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
					users = []auth.User{}
				}
				server = createAuthServer(tc.hostURL, tc.Name, tc.providerKind, currUser, users...)
			} else {
				currUser = auth.User{
					Username: tc.username,
					ApiToken: tc.apiToken,
				}
				server = createAuthServer(tc.hostURL, tc.Name, tc.providerKind, currUser, users...)
			}
			result, err := gits.CreateProvider(server, tc.git)
			if tc.wantError {
				assert.Error(t, err, "should fail to create provider")
				assert.Nil(t, result, "created provider should be nil")
			} else {
				assert.NoError(t, err, "should create provider without error")
				assert.NotNil(t, result, "created provider should not be nil")
				want := createGitProvider(t, tc.providerKind, server, tc.git)
				assertProvider(t, want, result)
			}
		})
	}
}

func assertProvider(t *testing.T, want gits.GitProvider, result gits.GitProvider) {
	assert.Equal(t, want.Kind(), result.Kind())
	assert.Equal(t, want.Server(), result.Server())
}
