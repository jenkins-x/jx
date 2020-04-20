// +build unit

package update

import (
	"testing"

	jenkinsio "github.com/jenkins-x/jx/pkg/apis/jenkins.io"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/gits"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetOrgOrUserFromOptions_orgIsSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:           "MyOrg",
		CommonOptions: &opts.CommonOptions{Username: "MyUser"},
	}
	owner := options.GetOrgOrUserFromOptions()
	assert.Equal(t, options.Org, owner, "The Owner should be the Org name")
}

func TestGetOrgOrUserFromOptions_orgNotSetUserIsSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:           "",
		CommonOptions: &opts.CommonOptions{Username: "MyUser"},
	}
	owner := options.GetOrgOrUserFromOptions()
	assert.Equal(t, options.Username, owner, "The Owner should be the Username")
}

func TestGetOrgOrUserFromOptions_orgNotSetUserNotSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:           "",
		CommonOptions: &opts.CommonOptions{Username: ""},
	}
	owner := options.GetOrgOrUserFromOptions()
	assert.Equal(t, "", owner, "The Owner should be empty")
}

func TestUpdateWebhookForSourceRepository(t *testing.T) {
	t.Parallel()
	var err error
	org := "org"
	repo := "myRepo"
	username := "testOrgName"
	o := opts.NewCommonOptionsWithFactory(fake.NewFakeFactory())
	testhelpers.ConfigureTestOptions(&o, gits.NewGitFake(), helm_test.NewMockHelmer())
	fakeRepo, _ := gits.NewFakeRepository(org, repo, nil, nil)
	fakeGitProvider := gits.NewFakeProvider(fakeRepo)
	fakeGitProvider.User.Username = username
	o.SetFakeGitProvider(fakeGitProvider)
	updateWebhooksOptions := &UpdateWebhooksOptions{
		CommonOptions: &o,
	}
	updateWebhooksOptions.DryRun = true
	sr2, envMap2 := getSourceAndEnv(false)
	registered, err2 := updateWebhooksOptions.UpdateWebhookForSourceRepository(sr2, envMap2, err, "webhookURL", "emptyToken")
	assert.True(t, registered, "A webhook was not registered for the environment environment")
	assert.Nil(t, err2, "An error was returned trying to register a webhook")
}

func TestUpdateWebhookForSourceRepository_IgnoreRemoteEnv(t *testing.T) {
	t.Parallel()
	var err error
	updateWebhooksOptions := &UpdateWebhooksOptions{
		CommonOptions: &opts.CommonOptions{},
	}
	sr1, envMap1 := getSourceAndEnv(true)
	updateWebhooksOptions.DryRun = true
	registered, err := updateWebhooksOptions.UpdateWebhookForSourceRepository(sr1, envMap1, err, "webhookURL", "emptyToken")
	assert.Nil(t, err, "An error was returned trying to register a webhook")
	assert.False(t, registered, "A remote environment should not have webhooks registered for the dev environment")
}

func getSourceAndEnv(remote bool) (*v1.SourceRepository, map[string]*v1.Environment) {
	provider := "https://github.com"
	providerName := "github"
	org := "org"
	repo := "myRepo"
	envName := "test-env"
	branch := "my-branch"

	sr := &v1.SourceRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SourceRepository",
			APIVersion: jenkinsio.GroupName + "/" + jenkinsio.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: repo,
		},
		Spec: v1.SourceRepositorySpec{
			Description:  "description",
			Org:          org,
			Provider:     provider,
			ProviderName: providerName,
			Repo:         repo,
		},
	}

	env := &v1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name: envName,
		},
		Spec: v1.EnvironmentSpec{
			Kind:              v1.EnvironmentKindTypePermanent,
			PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
			Order:             999,
			RemoteCluster:     remote,
			Source: v1.EnvironmentRepository{
				Kind: v1.EnvironmentRepositoryTypeGit,
				URL:  provider + "/" + org + "/" + repo + ".git",
				Ref:  branch,
			},
		},
	}
	envMap := make(map[string]*v1.Environment)
	envMap[envName] = env
	return sr, envMap
}
