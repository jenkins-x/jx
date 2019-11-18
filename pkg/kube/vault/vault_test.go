package vault_test

import (
	"testing"

	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"

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
			vaultCRD, err := vault.PrepareGKEVaultCRD(client, tc.name, tc.namespace, tc.images, tc.secretName,
				tc.gcpConfig, tc.authServiceAccount, tc.namespace, tc.secretsPathPrefix)

			validateVault(err, vaultCRD, &tc.VaultTestCase, t, client)
		})
	}
}

func TestCreateAWSVault(t *testing.T) {
	client := k8sfake.NewSimpleClientset()

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
		vaultCRD, err := vault.PrepareAWSVaultCRD(client, tc.name, tc.namespace, tc.images, tc.secretName,
			awsConfig, tc.authServiceAccount, tc.namespace, tc.secretsPathPrefix)

		validateVault(err, vaultCRD, &tc, t, client)
	})
}

func validateVault(err error, vaultCRD *v1alpha1.Vault, tc *VaultTestCase, t *testing.T, client *k8sfake.Clientset) {
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
