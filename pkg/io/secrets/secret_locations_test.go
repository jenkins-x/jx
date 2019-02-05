package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const ns = "Galaxy"

func TestUseVaultForSecrets(t *testing.T) {
	t.Parallel()

	kubeClient := createMockCluster()
	secretLocation := NewSecretLocation(kubeClient, ns)

	err := secretLocation.SetLocation(VaultLocationKind)
	assert.NoError(t, err)

	// Test we have actually added the item to the configmap
	configMap, err := kubeClient.Core().ConfigMaps(ns).Get("jx-install-config", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, string(VaultLocationKind), configMap.Data[SecretsLocationKey])
	// Test we haven't overwritten the configmap
	assert.Equal(t, "two", configMap.Data["one"])
	assert.Equal(t, VaultLocationKind, secretLocation.Location())

	err = secretLocation.SetLocation(FileSystemLocationKind)
	assert.NoError(t, err)

	configMap, err = kubeClient.Core().ConfigMaps(ns).Get("jx-install-config", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, string(FileSystemLocationKind), configMap.Data[SecretsLocationKey])
	// Test we haven't overwritten the configmap
	assert.Equal(t, "two", configMap.Data["one"])
	assert.NotEqual(t, VaultLocationKind, secretLocation.Location())
}

func TestUseVaultForSecrets_NoJxInstallConfigMap(t *testing.T) {
	t.Parallel()

	kubeClient := fake.NewSimpleClientset()
	secretLocation := NewSecretLocation(kubeClient, ns)

	err := secretLocation.SetLocation(VaultLocationKind)
	assert.NoError(t, err)

	// Test we have actually added the item to the configmap
	configMap, err := kubeClient.Core().ConfigMaps(ns).Get("jx-install-config", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, string(VaultLocationKind), configMap.Data[SecretsLocationKey])
	assert.Equal(t, VaultLocationKind, secretLocation.Location())

	err = secretLocation.SetLocation(FileSystemLocationKind)
	assert.NoError(t, err)

	configMap, err = kubeClient.Core().ConfigMaps(ns).Get("jx-install-config", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, string(FileSystemLocationKind), configMap.Data[SecretsLocationKey])
	assert.NotEqual(t, VaultLocationKind, secretLocation.Location())
}

func createMockCluster() *fake.Clientset {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jx-install-config",
			Namespace: ns,
		},
		Data: map[string]string{"one": "two"},
	}
	kubeClient := fake.NewSimpleClientset(namespace, configMap)
	return kubeClient
}
