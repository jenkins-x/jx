// +build unit

package step_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
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

func TestCreateBucketHTTPFn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		gitURL              string
		expectedTokenPrefix string
		expectedHeader      string
		expectedHeaderValue string
	}{
		{
			name:                "bitbucketserver",
			gitURL:              "bitbucket.example.com/scm/some-org/some-proj.git",
			expectedTokenPrefix: "test:test",
		},
		{
			name:                "github",
			gitURL:              "raw.githubusercontent.com/jenkins-x/environment-tekton-weasel-dev/master/OWNERS",
			expectedTokenPrefix: "test",
		},
		{
			name:                "gitlab",
			gitURL:              "gitlab.com/api/v4/projects/jxbdd%2Fenvironment-pr-751-6-lh-bdd-gl-dev/repository/files/jenkins-x%2Flogs%2Fjxbdd%2Fenvironment-pr-751-6-lh-bdd-gl-dev%2FPR-1%2F1.log/raw?ref=gh-pages",
			expectedHeader:      "PRIVATE-TOKEN",
			expectedHeaderValue: "test",
		},
		{
			name:                "ghe",
			gitURL:              "github.something.com/raw/foo/bar/branch/blah.log",
			expectedTokenPrefix: "test",
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

			req, err := http.NewRequest("GET", "https://"+tc.gitURL, nil)
			assert.NoError(t, err)

			httpFn := step.CreateBucketHTTPFn(authCfgSvc)

			bucketURL, headerFn, err := httpFn("https://" + tc.gitURL)

			assert.NoError(t, err)
			expectedURL := fmt.Sprintf("https://%s@%s", tc.expectedTokenPrefix, tc.gitURL)
			if tc.expectedTokenPrefix == "" {
				expectedURL = fmt.Sprintf("https://%s", tc.gitURL)
			}
			assert.Equal(t, expectedURL, bucketURL)

			headerFn(req)

			if tc.expectedHeader != "" {
				assert.Equal(t, tc.expectedHeaderValue, req.Header.Get(tc.expectedHeader))
			}
		})
	}
}
