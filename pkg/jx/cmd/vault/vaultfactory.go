package vault

import (
	"errors"
	"fmt"
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"github.com/hashicorp/vault/api"
	"github.com/jenkins-x/jx/pkg/jx/cmd/common"
	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type VaultClientFactory struct {
	Options          common.NewCommonOptionsInterface
	Selector         VaultSelector
	kubeClient       kubernetes.Interface
	defaultNamespace string
}

func NewVaultClientFactory(options common.NewCommonOptionsInterface) (VaultClientFactory, error) {
	factory := VaultClientFactory{
		Options: options,
	}
	var err error
	factory.kubeClient, factory.defaultNamespace, err = options.KubeClient()
	if err != nil {
		return factory, err
	}
	factory.Selector, err = NewVaultSelector(options)
	if err != nil {
		return factory, err
	}
	return factory, nil
}

// NewVaultClient creates a new api.Client
// if namespace is nil, then the default namespace of the factory will be used
func (v VaultClientFactory) NewVaultClient(namespace string) (*api.Client, error) {
	config, jwt, role, err := v.GetConfigData(namespace)
	if err != nil {
		return nil, err
	}
	vaultClient, err := api.NewClient(config)
	token, err := getTokenFromVault(role, jwt, vaultClient)
	vaultClient.SetToken(token)
	return vaultClient, nil
}

// GetConfigData generates the information necessary to configure an api.Client object
// Returns the api.Config object, the JWT needed to create the auth user in vault, and an error if present
func (v *VaultClientFactory) GetConfigData(namespace string) (config *api.Config, jwt string, saName string, err error) {
	if namespace == "" {
		namespace = v.defaultNamespace
	}
	vlt, err := v.Selector.GetVault(namespace)
	if err != nil {
		return nil, "", "", err
	}

	serviceAccount, err := v.getServiceAccountFromVault(vlt)
	secret, err := v.getSecretFromServiceAccount(serviceAccount, vlt.Namespace)

	return &api.Config{Address: vlt.URL}, getJWTFromSecret(secret), serviceAccount.Name, nil
}

func (v *VaultClientFactory) getServiceAccountFromVault(vault *kube.Vault) (*v1.ServiceAccount, error) {
	return v.kubeClient.CoreV1().ServiceAccounts(vault.Namespace).Get(vault.AuthServiceAccountName, meta_v1.GetOptions{})
}

func (v *VaultClientFactory) getSecretFromServiceAccount(sa *v1.ServiceAccount, namespace string) (*v1.Secret, error) {
	secretName := sa.Secrets[0].Name
	return v.kubeClient.CoreV1().Secrets(namespace).Get(secretName, meta_v1.GetOptions{})
}

func getJWTFromSecret(secret *v1.Secret) string {
	return string(secret.Data["token"])
}

func getTokenFromVault(role string, jwt string, vaultClient *api.Client) (string, error) {
	m := map[string]interface{}{
		"jwt":  jwt,
		"role": role,
	}
	sec, err := vaultClient.Logical().Write("/auth/kubernetes/login", m)
	return sec.Auth.ClientToken, err
}

type vaultSelectorImpl struct {
	vaultOperatorClient versioned.Interface
	kubeClient          kubernetes.Interface
}

func NewVaultSelector(o common.NewCommonOptionsInterface) (VaultSelector, error) {
	operator, err := o.VaultOperatorClient()
	if err != nil {
		return nil, err
	}
	kubeclient, _, err := o.KubeClient()
	if err != nil {
		return nil, err
	}
	v := vaultSelectorImpl{
		vaultOperatorClient: operator,
		kubeClient:          kubeclient,
	}
	return v, nil
}

func (v vaultSelectorImpl) GetVault(namespace string) (*kube.Vault, error) {
	vaults, err := kube.GetVaults(v.kubeClient, v.vaultOperatorClient, namespace)
	if err != nil {
		return nil, err
	} else if len(vaults) == 0 {
		return nil, errors.New(fmt.Sprintf("no vaults found in namespace '%s'", namespace))
	}
	if len(vaults) > 1 {
		return selectVault(vaults)
	}
	return vaults[0], nil
}

func selectVault(vaults []*kube.Vault) (*kube.Vault, error) {
	return vaults[0], nil // TODO - actually select the vault
}
