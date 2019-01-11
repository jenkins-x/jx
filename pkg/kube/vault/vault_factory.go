package vault

import (
	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"github.com/hashicorp/vault/api"
	"github.com/jenkins-x/jx/pkg/jx/cmd/common"
	"github.com/jenkins-x/jx/pkg/kube/serviceaccount"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// VaultClientFactory keeps the configuration required to build a new VaultClientFactory
type VaultClientFactory struct {
	Options          common.OptionsInterface
	Selector         Selector
	kubeClient       kubernetes.Interface
	defaultNamespace string
}

// NewInteractiveVaultClientFactory creates a VaultClientFactory that allows the user to pick vaults if necessary
func NewInteractiveVaultClientFactory(options common.OptionsInterface) (*VaultClientFactory, error) {
	factory := &VaultClientFactory{
		Options: options,
	}
	var err error
	factory.kubeClient, factory.defaultNamespace, err = options.KubeClientAndNamespace()
	if err != nil {
		return factory, err
	}
	factory.Selector, err = NewVaultSelector(options)
	if err != nil {
		return factory, err
	}
	return factory, nil
}

// NewVaultClientFactory Creates a new VaultClientFactory with different options to the above. It doesnt' have CLI support so
// will fail if it needs interactive input (unlikely)
func NewVaultClientFactory(kubeClient kubernetes.Interface, vaultOperatorClient versioned.Interface, defaultNamespace string) (*VaultClientFactory, error) {
	return &VaultClientFactory{
		kubeClient:       kubeClient,
		defaultNamespace: defaultNamespace,
		Selector: &vaultSelector{
			kubeClient:          kubeClient,
			vaultOperatorClient: vaultOperatorClient,
		},
	}, nil
}

// NewVaultClient creates a new api.Client
// if namespace is nil, then the default namespace of the factory will be used
// if the name is nil, and only one vault is found, then that vault will be used. Otherwise the user will be prompted to
// select a vault for the client.
func (v *VaultClientFactory) NewVaultClient(name string, namespace string) (*api.Client, error) {
	config, jwt, role, err := v.GetConfigData(name, namespace)
	if err != nil {
		return nil, err
	}
	vaultClient, err := api.NewClient(config)
	token, err := getTokenFromVault(role, jwt, vaultClient)
	if err != nil {
		return nil, errors.Wrapf(err, "getting Vault signin token")
	}
	vaultClient.SetToken(token)
	return vaultClient, nil
}

// GetConfigData generates the information necessary to configure an api.Client object
// Returns the api.Config object, the JWT needed to create the auth user in vault, and an error if present
func (v *VaultClientFactory) GetConfigData(name string, namespace string) (config *api.Config, jwt string, saName string, err error) {
	if namespace == "" {
		namespace = v.defaultNamespace
	}
	vlt, err := v.Selector.GetVault(name, namespace)
	if err != nil {
		return nil, "", "", err
	}

	serviceAccount, err := v.getServiceAccountFromVault(vlt)
	token, err := serviceaccount.GetServiceAccountToken(v.kubeClient, namespace, serviceAccount.Name)

	return &api.Config{Address: vlt.URL}, token, serviceAccount.Name, err
}

func (v *VaultClientFactory) getServiceAccountFromVault(vault *Vault) (*v1.ServiceAccount, error) {
	return v.kubeClient.CoreV1().ServiceAccounts(vault.Namespace).Get(vault.AuthServiceAccountName, meta_v1.GetOptions{})
}

func getTokenFromVault(role string, jwt string, vaultClient *api.Client) (string, error) {
	if role == "" {
		return "", errors.New("role cannot be empty")
	}
	if jwt == "" {
		return "", errors.New("JWT cannot be empty empty")
	}
	m := map[string]interface{}{
		"jwt":  jwt,
		"role": role,
	}
	sec, err := vaultClient.Logical().Write("/auth/kubernetes/login", m)
	if err != nil {
		return "", err
	}
	return sec.Auth.ClientToken, err
}
