package git_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/cmd/step/git"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepGitCredentials(t *testing.T) {
	t.Parallel()
	kind1 := gits.KindGitHub
	scheme1 := "https://"
	host1 := "github.com"
	user1 := "jstrachan"
	pwd1 := "lovelyLager"

	kind2 := gits.KindGitHub
	scheme2 := "http://"
	host2 := "github.beescloud.com"
	user2 := "rawlingsj"
	pwd2 := "glassOfNice"

	expected := createGitCredentialLine(scheme1, host1, user1, pwd1) +
		createGitCredentialLine(scheme2, host2, user2, pwd2)

	tmpDir, err := ioutil.TempDir("", "gitcredentials")
	if err != nil {
		require.NoError(t, err, "should create a temporary dir")
	}
	defer os.RemoveAll(tmpDir)
	outFile := filepath.Join(tmpDir, "credentials")
	options := &git.StepGitCredentialsOptions{
		OutputFile: outFile,
	}

	authSvc := auth.NewMemoryAuthConfigService()
	cfg := auth.AuthConfig{
		Servers: []*auth.AuthServer{
			{
				URL: scheme1 + host1,
				Users: []*auth.UserAuth{
					{
						Username: user1,
						ApiToken: pwd1,
					},
				},
				Name:        kind1,
				Kind:        kind1,
				CurrentUser: user1,
			},
			{
				URL: scheme2 + host2,
				Users: []*auth.UserAuth{
					{
						Username: user2,
						ApiToken: pwd2,
					},
				},
				Name:        kind2,
				Kind:        kind2,
				CurrentUser: user2,
			},
		},
	}
	authSvc.SetConfig(&cfg)

	err = options.CreateGitCredentialsFile(outFile, authSvc)
	assert.NoError(t, err, "should create the git credentials file without error")

	actual, err := ioutil.ReadFile(outFile)
	assert.NoError(t, err, "should read the git credentials from file")
	assert.EqualValues(t, expected, string(actual), "generated git credentials file")
}

func TestStepGitCredentialsForGitHubAppSecrets(t *testing.T) {
	t.Parallel()
	kind := gits.KindGitHub
	scheme := "https://"
	host := "github.com"
	botUser := "jenkins-x-bot"

	owner1 := "rawlingsj"
	pwd1 := "lovelyLager"
	owner2 := "jstrachan"
	pwd2 := "glassOfNice"
	owner3 := "abayer"
	pwd3 := "edam"

	expected := createGitCredentialLine(scheme, host, botUser, pwd2)

	tmpDir, err := ioutil.TempDir("", "gitcredentials")
	if err != nil {
		require.NoError(t, err, "should create a temporary dir")
	}
	defer os.RemoveAll(tmpDir)
	outFile := filepath.Join(tmpDir, "credentials")
	options := &git.StepGitCredentialsOptions{
		OutputFile:     outFile,
		GitHubAppOwner: owner2,
		GitKind:        kind,
	}

	authSvc := auth.NewMemoryAuthConfigService()
	cfg := auth.AuthConfig{
		Servers: []*auth.AuthServer{
			{
				URL: scheme + host,
				Users: []*auth.UserAuth{
					{
						GithubAppOwner: owner1,
						Username:       botUser,
						ApiToken:       pwd1,
					},
					{
						GithubAppOwner: owner2,
						Username:       botUser,
						ApiToken:       pwd2,
					},
					{
						GithubAppOwner: owner3,
						Username:       botUser,
						ApiToken:       pwd3,
					},
				},
				Name: "gh",
				Kind: kind,
			},
		},
	}
	authSvc.SetConfig(&cfg)

	err = options.CreateGitCredentialsFile(outFile, authSvc)
	assert.NoError(t, err, "should create the git credentials file without error")

	actual, err := ioutil.ReadFile(outFile)
	assert.NoError(t, err, "should read the git credentials from file")
	assert.EqualValues(t, expected, string(actual), "generated git credentials file")
}

func createGitCredentialLine(scheme string, host string, user string, pwd string) string {
	answer := scheme + user + ":" + pwd + "@" + host + "\n"
	if scheme == "http://" {
		scheme = "https://"
	}
	answer += scheme + user + ":" + pwd + "@" + host + "\n"
	return answer
}
