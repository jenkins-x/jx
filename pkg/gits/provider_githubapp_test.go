package gits_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/gits"
	mocks "github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/testkube"
	utiltests "github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateGitProviderFromURLForGitHubApp(t *testing.T) {
	t.Parallel()
	utiltests.SkipForWindows(t, "go-expect does not work on Windows")

	testData := []struct {
		ghOwner  string
		apiToken string
	}{
		{
			"owner1",
			"tokenOwner1",
		},
		{
			"owner2",
			"tokenOwner2",
		},
		{
			"owner3",
			"tokenOwner3",
		},
	}

	// lets assert that we can find tokens for github apps
	tests := []struct {
		ghOwner       string
		expectedToken string
		wantError     bool
	}{
		{
			"doesNotExist",
			"doesNotExist",
			true,
		},
		{
			"owner1",
			"tokenOwner1",
			false,
		},
		{
			"owner2",
			"tokenOwner2",
			false,
		},
		{
			"owner3",
			"tokenOwner3",
			false,
		},
	}

	gitServiceName := "gh"
	gitServerURL := "https://github.com"
	providerKind := "github"
	inCluster := true
	batchMode := true
	username := "jenkins-x[bot]"

	secretList := &corev1.SecretList{
		Items: []corev1.Secret{},
	}

	for _, td := range testData {
		secret := testkube.CreateTestPipelineGitSecret(providerKind, gitServiceName, gitServerURL, username, td.apiToken)
		secret.Labels[kube.LabelGithubAppOwner] = td.ghOwner
		secretList.Items = append(secretList.Items, secret)
	}

	o := &opts.CommonOptions{}
	testhelpers.ConfigureTestOptions(o, gits.NewGitCLI(), helm.NewHelmCLI("helm", helm.V2, "", true))

	fileName := "doesNotExist.yaml"

	authSvc, err := o.CreateGitAuthConfigServiceFromSecrets(fileName, secretList, true)
	require.NoError(t, err, "could not create AuthConfigService from secrets")

	handles := util.IOFileHandles{}

	for _, tt := range tests {
		ghOwner := tt.ghOwner
		git := mocks.NewMockGitter()
		result, err := gits.CreateProviderForURL(inCluster, authSvc, providerKind, gitServerURL, ghOwner, git, batchMode, handles)

		if tt.wantError {
			assert.Error(t, err, "should fail to create provider for owner %s", ghOwner)
			assert.Nil(t, result, "created provider should be nil for owner %s", ghOwner)
			t.Logf("got expected error for owner %s: %s", ghOwner, err.Error())
		} else {
			assert.NoError(t, err, "should create provider without error for owner %s", ghOwner)
			assert.NotNil(t, result, "created provider should not be nil for owner %s", ghOwner)

			userAuth := result.UserAuth()
			assert.NotNil(t, userAuth, "provider userAuth should not be nil for owner %s", ghOwner)
			assert.Equal(t, tt.expectedToken, userAuth.ApiToken, "provider userAuth.ApiToken for owner %s", ghOwner)
			t.Logf("owner %s got GitHub App token %s", ghOwner, userAuth.ApiToken)
		}
	}
}
