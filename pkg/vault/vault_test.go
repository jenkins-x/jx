package vault_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/vault"

	fakevaultclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestCreateVault(t *testing.T) {

	client := k8sfake.NewSimpleClientset()
	vaultclient := fakevaultclient.NewSimpleClientset()

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
			err := vault.CreateVault(client, vaultclient, tc.name, tc.namespace, tc.gcpSecretName,
				tc.gcpConfig, tc.authServiceAccount, tc.namespace, tc.secretsPathPrefix)
			if tc.err {
				assert.Error(t, err, "should create vault with an error")
			} else {
				assert.NoError(t, err, "should create vault without an error")
			}

			vault, err := vaultclient.Vault().Vaults(tc.namespace).Get(tc.name, metav1.GetOptions{})
			assert.NoError(t, err, "should retrieve created vault without an error")
			assert.NotNil(t, vault, "created vault should not be nil")
			sa, err := client.CoreV1().ServiceAccounts(tc.namespace).Get(tc.name, metav1.GetOptions{})
			assert.NoError(t, err, "should retrieve vault service account without error")
			assert.NotNil(t, sa, "created vault service account should not be nil")
			role, err := client.RbacV1().ClusterRoles().Get("vault-auth", metav1.GetOptions{})
			assert.NoError(t, err, "should retrieve vault cluster role without error")
			assert.NotNil(t, role, "created vault cluster role should not be nil")
			rb, err := client.RbacV1().ClusterRoleBindings().Get(tc.name, metav1.GetOptions{})
			assert.NoError(t, err, "should retrieve vault cluster role binding without error")
			assert.NotNil(t, rb, "created vault cluster role binding should not be nil")
		})
	}
}
