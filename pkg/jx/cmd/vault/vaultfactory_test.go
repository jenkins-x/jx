package vault_test

import (
	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	cmdMocks "github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd/vault"
	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/client-go/kubernetes"

	. "github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestGetConfigData(t *testing.T) {
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t)

	vaultName, namespace := "myVault", "myVaultNamespace"
	createMockedVault(vaultName, namespace, "foo.bar", "myJWT", vaultOperatorClient, kubeClient)

	// Invoke the function under test
	config, jwt, saName, err := factory.GetConfigData(namespace)

	assert.Equal(t, "http://foo.bar", config.Address)
	assert.Equal(t, "myJWT", jwt)
	assert.Equal(t, "myVault-auth-sa", saName)
	assert.NoError(t, err)
}

func TestGetConfigData_DefaultNamespacesUsed(t *testing.T) {
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t)

	vaultName, namespace := "myVault", "jx" // "jx" is the default namespace used by the kubeClient
	createMockedVault(vaultName, namespace, "foo.bar", "myJWT", vaultOperatorClient, kubeClient)

	// Invoke the function under test
	config, jwt, saName, err := factory.GetConfigData("")

	assert.Equal(t, "http://foo.bar", config.Address)
	assert.Equal(t, "myJWT", jwt)
	assert.Equal(t, "myVault-auth-sa", saName)
	assert.NoError(t, err)
}

func TestGetConfigData_ErrorsWhenNoVaultsInNamespace(t *testing.T) {
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t)

	vaultName, namespace := "myVault", "myVaultNamespace"
	createMockedVault(vaultName, namespace, "foo.bar", "myJWT", vaultOperatorClient, kubeClient)

	// Invoke the function under test
	config, jwt, saName, err := factory.GetConfigData("Nothing In This Namespace")

	assert.Nil(t, config)
	assert.Empty(t, jwt)
	assert.Empty(t, saName)
	assert.EqualError(t, err, "no vaults found in namespace 'Nothing In This Namespace'")
}

func TestGetConfigData_ConfigUsedFromVaultSelector(t *testing.T) {
	// Two vaults are configured in the same namespace, the user specifies one with the -m flag
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t)

	namespace := "myVaultNamespace"
	_ = createMockedVault("vault1", namespace, "one.ah.ah.ah", "count", vaultOperatorClient, kubeClient)
	vault2 := createMockedVault("vault2", namespace, "two.ah.ah.ah", "von-count", vaultOperatorClient, kubeClient)

	// Create a mock Selector that just returns the second vault
	factory.Selector = PredefinedVaultSelector{vaultToReturn: vault2, url: "http://two.ah.ah.ah"}

	// Invoke the function under test
	config, jwt, saName, err := factory.GetConfigData(namespace)

	assert.Equal(t, "http://two.ah.ah.ah", config.Address)
	assert.Equal(t, "von-count", jwt)
	assert.Equal(t, "vault2-auth-sa", saName)
	assert.NoError(t, err)
}

func setupMocks(t *testing.T) (*fake.Clientset, vault.VaultClientFactory, error, kubernetes.Interface) {
	options := cmd.NewCommonOptions("", cmdMocks.NewMockFactory())
	cmd.ConfigureTestOptions(&options, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
	vaultOperatorClient := fake.NewSimpleClientset()
	When(options.Factory.CreateVaultOperatorClient()).ThenReturn(vaultOperatorClient, nil)
	f, err := vault.NewVaultClientFactory(&options)
	kubeClient, _, err := options.KubeClient()
	assert.NoError(t, err)
	return vaultOperatorClient, f, err, kubeClient
}

func createMockedVault(vaultName string, namespace string, vaultUrl string, jwt string,
	vaultOperatorClient *fake.Clientset, kubeClient kubernetes.Interface) v1alpha1.Vault {

	v := v1alpha1.Vault{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vaultName,
			Namespace: namespace,
		},
	}
	secretName := vaultName + "-secret"
	_, _ = vaultOperatorClient.Vault().Vaults(namespace).Create(&v)
	_, _ = kubeClient.CoreV1().ServiceAccounts(namespace).Create(&v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: vaultName + "-auth-sa"},
		Secrets:    []v1.ObjectReference{{Name: secretName}},
	})
	_, _ = kubeClient.CoreV1().Services(namespace).Create(&v1.Service{ObjectMeta: metav1.ObjectMeta{Name: vaultName}})
	_, _ = kubeClient.ExtensionsV1beta1().Ingresses(namespace).Create(&v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: vaultName},
		Spec:       v1beta1.IngressSpec{Rules: []v1beta1.IngressRule{{Host: vaultUrl}}},
	})
	_, _ = kubeClient.CoreV1().Secrets(namespace).Create(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName},
		Data:       map[string][]byte{"token": []byte(jwt)},
	})
	return v
}

// PredefinedVaultSelector is a dummy Selector that returns a pre-defined vault
type PredefinedVaultSelector struct {
	vaultToReturn v1alpha1.Vault
	url           string
}

func (p PredefinedVaultSelector) GetVault(namespaces string) (*kube.Vault, error) {
	return &kube.Vault{
		Name:                   p.vaultToReturn.Name,
		Namespace:              p.vaultToReturn.Namespace,
		AuthServiceAccountName: p.vaultToReturn.Name + "-auth-sa",
		URL:                    p.url,
	}, nil
}
