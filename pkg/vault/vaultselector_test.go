package vault_test

import (
	"github.com/Netflix/go-expect"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/stretchr/testify/assert"
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
	console := tests.NewTerminal(t)
	vaultOperatorClient, factory, err, kubeClient := setupMocks(t, &console.Stdio)
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

	console.Close()
	<-donec

	// Dump the terminal's screen.
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))

	assert.Equal(t, "vault2", vault.Name)
	assert.Equal(t, "myVaultNamespace", vault.Namespace)
	assert.Equal(t, "http://two.ah.ah.ah", vault.URL)
	assert.Equal(t, "vault2-auth-sa", vault.AuthServiceAccountName)
	assert.NoError(t, err)
}
