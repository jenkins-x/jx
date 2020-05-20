// +build unit

package step_test

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/auth"
	"github.com/jenkins-x/jx/v2/pkg/cmd/step"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/secreturl/fakevault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

const secretName = kube.SecretJenkinsPipelineGitCredentials + "github-ghe"

func TestGetTokenForGitURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		gitURL        string
		expectedToken string
	}{
		{
			name:          "bitbucketserver",
			gitURL:        "https://bitbucket.example.com/scm/some-org/some-proj.git",
			expectedToken: "test:test",
		},
		{
			name:          "github",
			gitURL:        "https://raw.githubusercontent.com/jenkins-x/environment-tekton-weasel-dev/master/OWNERS",
			expectedToken: "test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ns := "jx"

			authFile := path.Join("test_data", "step_unstash", tc.name+".yaml")
			data, err := ioutil.ReadFile(authFile)
			require.NoError(t, err)

			var expectedAuthConfig auth.AuthConfig
			err = yaml.Unmarshal(data, &expectedAuthConfig)
			require.NoError(t, err)

			namespace := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			}
			config := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jx-auth",
					Namespace: ns,
					Labels: map[string]string{
						"jenkins.io/config-type": "auth",
					},
				},
				Data: map[string]string{
					secretName: string(data),
				},
			}
			kubeClient := fake.NewSimpleClientset(namespace, config)
			configMapInterface := kubeClient.CoreV1().ConfigMaps(ns)

			vaultClient := fakevault.NewFakeClient()
			_, err = vaultClient.Write("test-cluster/pipelineUser", map[string]interface{}{"token": "test"})
			require.NoError(t, err)
			expectedAuthConfig.Servers[0].Users[0].ApiToken = "test"

			authCfgSvc := auth.NewConfigmapVaultAuthConfigService(secretName, configMapInterface, vaultClient)
			_, err = authCfgSvc.LoadConfig()
			assert.NoError(t, err)

			gitToken, err := step.GetTokenForGitURL(authCfgSvc, tc.gitURL)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedToken, gitToken)
		})
	}
}
