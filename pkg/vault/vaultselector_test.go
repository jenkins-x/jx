package vault_test

import (
	"github.com/Netflix/go-expect"
	"github.com/banzaicloud/bank-vaults/operator/pkg/apis/vault/v1alpha1"
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	cmdMocks "github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/vault"
	. "github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"testing"
)

func Test_GetVault_DoesNotPromptUserIfOnlyOneVaultInNamespace(t *testing.T) {
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t, nil)
	createMockedVault("myVault", "myVaultNamespace", "foo.bar", "myJWT", vaultOperatorClient, kubeClient)

	selector, err := vault.NewVaultSelector(factory.Options)

	vault, err := selector.GetVault("", "myVaultNamespace")

	assert.Equal(t, "myVault", vault.Name)
	assert.Equal(t, "myVaultNamespace", vault.Namespace)
	assert.Equal(t, "http://foo.bar", vault.URL)
	assert.Equal(t, "myVault-auth-sa", vault.AuthServiceAccountName)
	assert.NoError(t, err)
}

func Test_GetVault_ErrorsIfNoVaultsInNamespace(t *testing.T) {
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t, nil)
	createMockedVault("myVault", "myVaultNamespace", "foo.bar", "myJWT", vaultOperatorClient, kubeClient)

	selector, err := vault.NewVaultSelector(factory.Options)

	vault, err := selector.GetVault("", "Nothing Here Jim")

	assert.Nil(t, vault)
	assert.EqualError(t, err, "no vaults found in namespace 'Nothing Here Jim'")
}

func Test_GetVault_ErrorsIfRequestedVaultDoesNotExist(t *testing.T) {
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t, nil)
	createMockedVault("myVault", "myVaultNamespace", "foo.bar", "myJWT", vaultOperatorClient, kubeClient)

	selector, err := vault.NewVaultSelector(factory.Options)

	vault, err := selector.GetVault("NoVaultHere", "myVaultNamespace")

	assert.Nil(t, vault)
	assert.EqualError(t, err, "vault 'NoVaultHere' not found in namespace 'myVaultNamespace'")
}

func Test_GetVault_GetExplicitVaultSucceedsWhenTwoVaultsAreDefined(t *testing.T) {
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t, nil)
	createMockedVault("vault1", "myVaultNamespace", "one.ah.ah.ah", "Count", vaultOperatorClient, kubeClient)
	createMockedVault("vault2", "myVaultNamespace", "two.ah.ah.ah", "Von-Count", vaultOperatorClient, kubeClient)

	selector, err := vault.NewVaultSelector(factory.Options)

	vault, err := selector.GetVault("vault2", "myVaultNamespace")

	assert.Equal(t, "vault2", vault.Name)
	assert.Equal(t, "myVaultNamespace", vault.Namespace)
	assert.Equal(t, "http://two.ah.ah.ah", vault.URL)
	assert.Equal(t, "vault2-auth-sa", vault.AuthServiceAccountName)
	assert.NoError(t, err)
}

func Test_GetVault_PromptsUserIfMoreThanOneVaultInNamespace(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")

	// mock terminal
	console, state, term := tests.NewTerminal(t)
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t, term)
	createMockedVault("vault1", "myVaultNamespace", "one.ah.ah.ah", "Count", vaultOperatorClient, kubeClient)
	createMockedVault("vault2", "myVaultNamespace", "two.ah.ah.ah", "Von-Count", vaultOperatorClient, kubeClient)

	selector, err := vault.NewVaultSelector(factory.Options)

	//Test interactive IO
	donec := make(chan struct{})
	go func() {
		defer close(donec)
		console.ExpectString("Select Vault:")
		console.SendLine("vault2")
		console.ExpectEOF()
	}()

	vault, err := selector.GetVault("", "myVaultNamespace")

	console.Tty().Close()
	<-donec

	// Dump the terminal's screen.
	t.Logf(expect.StripTrailingEmptyLines(state.String()))

	assert.Equal(t, "vault2", vault.Name)
	assert.Equal(t, "myVaultNamespace", vault.Namespace)
	assert.Equal(t, "http://two.ah.ah.ah", vault.URL)
	assert.Equal(t, "vault2-auth-sa", vault.AuthServiceAccountName)
	assert.NoError(t, err)

	//_ = selector
}

func setupMocks(t *testing.T, term *terminal.Stdio) (*fake.Clientset, vault.VaultClientFactory, error, kubernetes.Interface) {
	options := cmd.NewCommonOptions("", cmdMocks.NewMockFactory())
	if term != nil {
		options.In, options.Out, options.Err = term.In, term.Out, term.Err
	}
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
