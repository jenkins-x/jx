package create

import (
	"testing"

	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testNamespace = "test-ns"
	testVaultName = "test-vault"

	testSeal = vault.Seal{
		GcpCkms: &vault.GCPSealConfig{
			Credentials: "/",
			Project:     "acme",
			Region:      "secret",
			KeyRing:     "secret",
			CryptoKey:   "secret",
		},
	}

	testStorage = vault.Storage{
		GCS: &vault.GCSConfig{
			Bucket:    "my-gcs-bucket",
			HaEnabled: "true",
		},
	}
)

func Test_extractCloudProviderConfig_from_valid_vault_CRD_succeeds(t *testing.T) {
	vaultOperatorClient := fake.NewSimpleClientset()
	createTestVaultCRD(t, vaultOperatorClient, &testStorage, &testSeal)

	vaultCRD, err := vaultOperatorClient.VaultV1alpha1().Vaults(testNamespace).Get(testVaultName, v1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, vaultCRD)

	options := CreateVaultOptions{}
	cloudProviderConfig, err := options.extractCloudProviderConfig(vaultCRD)
	assert.NoError(t, err)
	assert.NotNil(t, cloudProviderConfig)
	assert.Equal(t, "my-gcs-bucket", cloudProviderConfig.Storage.GCS.Bucket)
	assert.Equal(t, "acme", cloudProviderConfig.Seal.GcpCkms.Project)
}

func Test_extractCloudProviderConfig_with_missing_storage_config_fails(t *testing.T) {
	vaultOperatorClient := fake.NewSimpleClientset()
	createTestVaultCRD(t, vaultOperatorClient, nil, &testSeal)

	vaultCRD, err := vaultOperatorClient.VaultV1alpha1().Vaults(testNamespace).Get(testVaultName, v1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, vaultCRD)

	options := CreateVaultOptions{}
	_, err = options.extractCloudProviderConfig(vaultCRD)
	assert.Error(t, err)
}

func Test_extractCloudProviderConfig_with_missing_seal_config_fails(t *testing.T) {
	vaultOperatorClient := fake.NewSimpleClientset()
	createTestVaultCRD(t, vaultOperatorClient, &testStorage, nil)

	vaultCRD, err := vaultOperatorClient.VaultV1alpha1().Vaults(testNamespace).Get(testVaultName, v1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, vaultCRD)

	options := CreateVaultOptions{}
	_, err = options.extractCloudProviderConfig(vaultCRD)
	assert.Error(t, err)
}

func createTestVaultCRD(t *testing.T, vaultOperatorClient *fake.Clientset, storage *vault.Storage, seal *vault.Seal) {
	vaultCRD := &v1alpha1.Vault{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Vault",
			APIVersion: "vault.banzaicloud.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testVaultName,
			Namespace: testNamespace,
		},
		Spec: v1alpha1.VaultSpec{
			Config: map[string]interface{}{},
		},
	}

	if storage != nil {
		vaultCRD.Spec.Config["storage"] = *storage
	}
	if seal != nil {
		vaultCRD.Spec.Config["seal"] = *seal
	}

	_, err := vaultOperatorClient.VaultV1alpha1().Vaults(testNamespace).Create(vaultCRD)
	assert.NoError(t, err)
}
