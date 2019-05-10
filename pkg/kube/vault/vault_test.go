package vault_test

import (
	"testing"

	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"

	fakevaultclient "github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

type VaultTestCase struct {
	name               string
	namespace          string
	images             map[string]string
	err                bool
	authServiceAccount string
	secretsPathPrefix  string
	secretName         string
}

func TestCreateGKEVault(t *testing.T) {

	client := k8sfake.NewSimpleClientset()
	vaultclient := fakevaultclient.NewSimpleClientset()

	tests := map[string]struct {
		VaultTestCase
		gcpConfig *vault.GCPConfig
	}{
		"create vault in GKE": {
			VaultTestCase: VaultTestCase{
				name:      "test-vault",
				namespace: "test-ns",
				images: map[string]string{
					vault.VaultImage:      vault.VaultImage + ":latest",
					vault.BankVaultsImage: vault.BankVaultsImage + ":latest",
				},
				authServiceAccount: "test-auth",
				secretsPathPrefix:  "test/*",
				err:                false,
				secretName:         "test-gcp",
			},
			gcpConfig: &vault.GCPConfig{
				ProjectId:   "test",
				KmsKeyring:  "test",
				KmsKey:      "test",
				KmsLocation: "test",
				GcsBucket:   "test",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := vault.CreateGKEVault(client, vaultclient, tc.name, tc.namespace, tc.images, tc.secretName,
				tc.gcpConfig, tc.authServiceAccount, tc.namespace, tc.secretsPathPrefix)

			validateVault(err, vaultclient, &tc.VaultTestCase, t, client)
		})
	}
}

func TestCreateAWSVault(t *testing.T) {

	client := k8sfake.NewSimpleClientset()
	vaultclient := fakevaultclient.NewSimpleClientset()

	tc := VaultTestCase{
		name:      "test-vault",
		namespace: "test-ns",
		images: map[string]string{
			vault.VaultImage:      vault.VaultImage + ":latest",
			vault.BankVaultsImage: vault.BankVaultsImage + ":latest",
		},
		authServiceAccount: "test-auth",
		secretsPathPrefix:  "test/*",
		err:                false,
		secretName:         "test-aws",
	}

	awsConfig := &vault.AWSConfig{
		AWSUnsealConfig: v1alpha1.AWSUnsealConfig{
			KMSKeyID:  "test",
			KMSRegion: "test",
			S3Bucket:  "test",
			S3Prefix:  "test",
			S3Region:  "test",
		},
		DynamoDBTable:   "test",
		DynamoDBRegion:  "test",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	}

	t.Run("create vault in AWS", func(t *testing.T) {
		err := vault.CreateAWSVault(client, vaultclient, tc.name, tc.namespace, tc.images, tc.secretName,
			awsConfig, tc.authServiceAccount, tc.namespace, tc.secretsPathPrefix)

		validateVault(err, vaultclient, &tc, t, client)
	})
}

func validateVault(err error, vaultclient *fakevaultclient.Clientset, tc *VaultTestCase, t *testing.T, client *k8sfake.Clientset) {
	if tc.err {
		assert.Error(t, err, "should create vault with an error")
	} else {
		assert.NoError(t, err, "should create vault without an error")
	}

	vaultCRD, err := vaultclient.Vault().Vaults(tc.namespace).Get(tc.name, metav1.GetOptions{})
	assert.NoError(t, err, "should retrieve created vault without an error")
	assert.NotNil(t, vaultCRD, "created vault should not be nil")
	sa, err := client.CoreV1().ServiceAccounts(tc.namespace).Get(tc.name, metav1.GetOptions{})
	assert.NoError(t, err, "should retrieve vault service account without error")
	assert.NotNil(t, sa, "created vault service account should not be nil")
	role, err := client.RbacV1().ClusterRoles().Get("vault-auth", metav1.GetOptions{})
	assert.NoError(t, err, "should retrieve vault cluster role without error")
	assert.NotNil(t, role, "created vault cluster role should not be nil")
	rb, err := client.RbacV1().ClusterRoleBindings().Get(tc.name, metav1.GetOptions{})
	assert.NoError(t, err, "should retrieve vault cluster role binding without error")
	assert.NotNil(t, rb, "created vault cluster role binding should not be nil")
}
