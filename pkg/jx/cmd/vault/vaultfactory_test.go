package vault_test

import (
	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	cmd_mocks "github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd/vault"
	"k8s.io/client-go/kubernetes"

	. "github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestGetConfigData(t *testing.T) {
	options := cmd.NewCommonOptions("", cmd_mocks.NewMockFactory())
	cmd.ConfigureTestOptions(&options, gits_test.NewMockGitter(), helm_test.NewMockHelmer())

	vaultOperatorClient := fake.NewSimpleClientset()
	When(options.Factory.CreateVaultOperatorClient()).ThenReturn(vaultOperatorClient, nil)
	f, err := vault.NewVaultClientFactory(&options)
	kubeClient, _, err := options.KubeClient()
	assert.NoError(t, err)
	vaultName, namespace := "myVault", "myVaultNamespace"
	v := v1alpha1.Vault{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vaultName,
			Namespace: namespace,
		},
	}
	createMockedVaultEnvironment(vaultName, namespace, "foo.bar", "myJWT", vaultOperatorClient, v, kubeClient)

	config, jwt, saName, err := f.GetConfigData(namespace)

	assert.Equal(t, "http://foo.bar", config.Address)
	assert.Equal(t, "myJWT", jwt)
	assert.Equal(t, "myVault-auth-sa", saName)
	assert.NoError(t, err)
}

func createMockedVaultEnvironment(vaultName string, namespace string, vaultUrl string, jwt string,
	vaultOperatorClient *fake.Clientset, v v1alpha1.Vault, kubeClient kubernetes.Interface) {
	secretName := "myVaultSecret"
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
}
