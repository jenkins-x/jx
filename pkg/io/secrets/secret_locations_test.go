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

	err := secretLocation.SetInVault(true)
	assert.NoError(t, err)

	// Test we have actually added the item to the configmap
	configMap, err := kubeClient.Core().ConfigMaps(ns).Get("jx-install-config", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "true", configMap.Data["useVaultForSecrets"])
	// Test we haven't overwritten the configmap
	assert.Equal(t, "two", configMap.Data["one"])
	assert.True(t, secretLocation.InVault())

	err = secretLocation.SetInVault(false)
	assert.NoError(t, err)

	configMap, err = kubeClient.Core().ConfigMaps(ns).Get("jx-install-config", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "", configMap.Data["useVaultForSecrets"])
	// Test we haven't overwritten the configmap
	assert.Equal(t, "two", configMap.Data["one"])
	assert.False(t, secretLocation.InVault())
}

func TestUseVaultForSecrets_NoJxInstallConfigMap(t *testing.T) {
	t.Parallel()

	kubeClient := fake.NewSimpleClientset()
	secretLocation := NewSecretLocation(kubeClient, ns)

	err := secretLocation.SetInVault(true)
	assert.NoError(t, err)

	// Test we have actually added the item to the configmap
	configMap, err := kubeClient.Core().ConfigMaps(ns).Get("jx-install-config", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "true", configMap.Data["useVaultForSecrets"])
	assert.True(t, secretLocation.InVault())

	err = secretLocation.SetInVault(false)
	assert.NoError(t, err)

	configMap, err = kubeClient.Core().ConfigMaps(ns).Get("jx-install-config", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "", configMap.Data["useVaultForSecrets"])
	assert.False(t, secretLocation.InVault())
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
