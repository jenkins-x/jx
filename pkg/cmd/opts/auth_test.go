package opts_test

import (
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/testkube"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

type gitTestData struct {
	Kind, Name, URL, User, Password string
}

func TestAuthLoadFromPipelineGitCredentials(t *testing.T) {
	t.Parallel()
	testData := []gitTestData{
		{
			gits.KindGitHub, "GitHub", "https://github.com", "jstrachan", "loverlyLarger",
		},
		{
			gits.KindGitHub, "GHE", "https://github.beescloud.com", "rawlingsj", "glassOfNice",
		},
	}

	secretList := &corev1.SecretList{
		Items: []corev1.Secret{},
	}

	for _, td := range testData {
		secretList.Items = append(secretList.Items, testkube.CreateTestPipelineGitSecret(td.Kind, td.Name, td.URL, td.User, td.Password))
	}

	o := &opts.CommonOptions{}
	testhelpers.ConfigureTestOptions(o, gits.NewGitCLI(), helm.NewHelmCLI("helm", helm.V2, "", true))

	fileName := "doesNotExist.yaml"

	authConfSvc, err := o.CreateGitAuthConfigServiceFromSecrets(fileName, secretList, true)
	assert.Nil(t, err, "Could not load Git Auth Config Service: %s", err)

	config := authConfSvc.Config()

	for _, svc := range config.Servers {
		tests.Debugf("Git URL %s has %d user(s)\n", svc.URL, len(svc.Users))
	}

	for _, td := range testData {
		url := td.URL
		user := td.User
		server := config.GetServer(url)
		assert.NotNil(t, server, "Could not find a Git server for url %s", url)
		assert.Equal(t, td.Name, server.Name)
		assert.Equal(t, td.Kind, server.Kind, "Kinds don't match for %s", url)
		assert.Equal(t, url, server.URL)

		userAuth := config.FindUserAuth(url, user)
		tests.Debugf("Server %s has %d users\n", url, len(server.Users))
		for _, u := range server.Users {
			tests.Debugf("Git URL %s has user %s/%s\n", url, u.Username, u.ApiToken)
		}

		assert.NotNil(t, userAuth, "No UserAuth found for url %s user %s", url, user)
		if userAuth != nil {
			assert.Equal(t, user, userAuth.Username)
			assert.Equal(t, td.Password, userAuth.ApiToken)
		}
	}
}
