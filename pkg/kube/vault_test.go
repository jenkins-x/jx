package kube_test

import (
	"github.com/jenkins-x/jx/pkg/vault"
	"testing"

	fakevaultclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateVault(t *testing.T) {
	client := fakevaultclient.NewSimpleClientset()

	tests := map[string]struct {
		name               string
		namespace          string
		gcpSecretName      string
		gcpConfig          *vault.GCPConfig
		authServiceAccount string
		secretsPathPrefix  string
		err                bool
	}{
		"create vault": {
			name:          "test-vault",
			namespace:     "test-ns",
			gcpSecretName: "test-gcp",
			gcpConfig: &vault.GCPConfig{
				ProjectId:   "test",
				KmsKeyring:  "test",
				KmsKey:      "test",
				KmsLocation: "test",
				GcsBucket:   "test",
			},
			authServiceAccount: "test-auth",
			secretsPathPrefix:  "test/*",
			err:                false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := vault.CreateVault(client, tc.name, tc.namespace, tc.gcpSecretName,
				tc.gcpConfig, tc.authServiceAccount, tc.namespace, tc.secretsPathPrefix)
			if tc.err {
				assert.Error(t, err, "should create vault with an error")
			} else {
				assert.NoError(t, err, "should create vault without an error")
			}

			vault, err := client.Vault().Vaults(tc.namespace).Get(tc.name, metav1.GetOptions{})
			assert.NoError(t, err, "should retrive created vault without an error")
			assert.NotNil(t, vault, "created vault should not be nil")
		})
	}
}
