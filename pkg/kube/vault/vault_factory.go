package vault

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jenkins-x/jx/v2/pkg/vault"

	"github.com/banzaicloud/bank-vaults/operator/pkg/client/clientset/versioned"
	"github.com/hashicorp/vault/api"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/kube/serviceaccount"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (

	// maxRetries controls the maximum number of time retry when 5xx error occurs. Default to 2 (for a total
	// of three retires)
	maxRetries = 2

	// healthReadyTimeout define the maximum duration to wait for vault to become initialized and unsealed
	healthRetryTimeout = 10 * time.Minute

	// healthInitialRetryDelay define the initial delay before starting the retries
	healthInitialRetryDelay = 10 * time.Second

	// authRetryTimeout define the maximum duration to wait for vault to authenticate
	authRetryTimeout = 1 * time.Minute

	// kvEngineConfigPath config path for KV secrets engine V2
	kvEngineConfigPath = "config"

	// kvEngineWriteCheckPath imaginary secret to check when the secrets engine is ready for write
	kvEngineWriteCheckPath = "data/jx-write-check"

	// kvEngineInitialRetryDelay define the initial delay before checking the kv engine configuration
	kvEngineInitialRetryDelay = 1 * time.Second

	// kvEngineRetryTimeout define the maximum duration to wait for KV engine to be properly configured
	kvEngineRetryTimeout = 5 * time.Minute
)

// OptionsInterface is an interface to allow passing around of a CommonOptions object without dependencies on the whole of the cmd package
type OptionsInterface interface {
	KubeClientAndNamespace() (kubernetes.Interface, string, error)
	VaultOperatorClient() (versioned.Interface, error)
	GetIn() terminal.FileReader
	GetOut() terminal.FileWriter
	GetErr() io.Writer
	GetIOFileHandles() util.IOFileHandles
}

// VaultClientFactory keeps the configuration required to build a new vault client factory
type VaultClientFactory struct {
	Options             OptionsInterface
	Selector            Selector
	kubeClient          kubernetes.Interface
	defaultNamespace    string
	DisableURLDiscovery bool
}

// NewInteractiveVaultClientFactory creates a VaultClientFactory that allows the user to pick vaults if necessary
func NewInteractiveVaultClientFactory(options OptionsInterface) (*VaultClientFactory, error) {
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

// NewVaultClientFactory creates a new VaultClientFactory with different options to the above. It doesnt' have CLI support so
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

// NewVaultClientFactoryWithoutSelector creates a new VaultClientFactory.
func NewVaultClientFactoryWithoutSelector(kubeClient kubernetes.Interface, defaultNamespace string) (*VaultClientFactory, error) {
	return &VaultClientFactory{
		kubeClient:       kubeClient,
		defaultNamespace: defaultNamespace,
	}, nil
}

// NewVaultClientFactoryWithSelector creates a new VaultClientFactory with a provided Selector.
// This allows to use an external Vault instance using the custom selector.
func NewVaultClientFactoryWithSelector(kubeClient kubernetes.Interface, selector Selector, defaultNamespace string) (*VaultClientFactory, error) {
	return &VaultClientFactory{
		kubeClient:       kubeClient,
		defaultNamespace: defaultNamespace,
		Selector:         selector,
	}, nil
}

// NewVaultClient creates a new api.Client
// if namespace is nil, then the default namespace of the factory will be used
// if the name is nil, and only one vault is found, then that vault will be used. Otherwise the user will be prompted to
// select a vault for the client.
func (v *VaultClientFactory) NewVaultClient(name string, namespace string, useIngressURL, insecureSSLWebhook bool) (*api.Client, error) {
	config, jwt, serviceAccountName, err := v.GetConfigData(name, namespace, useIngressURL, insecureSSLWebhook)
	if err != nil {
		return nil, err
	}

	vaultConfig := vault.Vault{
		Name:                   name,
		ServiceAccountName:     serviceAccountName,
		Namespace:              namespace,
		SecretEngineMountPoint: vault.DefaultKVEngineMountPoint,
		KubernetesAuthPath:     vault.DefaultKubernetesAuthPath,
	}

	return v.createClient(config, vaultConfig, jwt)
}

// NewVaultClientForURL creates a new Vault api.Client.
// If namespace is nil, then the default namespace of the factory will be used
func (v *VaultClientFactory) NewVaultClientForURL(vaultConfig vault.Vault, insecureSSLWebhook bool) (*api.Client, error) {
	serviceAccount, err := v.kubeClient.CoreV1().ServiceAccounts(vaultConfig.Namespace).Get(vaultConfig.ServiceAccountName, meta_v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get service account '%s'", vaultConfig.ServiceAccountName)
	}

	jwt, err := serviceaccount.GetServiceAccountToken(v.kubeClient, vaultConfig.Namespace, serviceAccount.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get service account token for '%s'", serviceAccount.Name)
	}

	config, err := v.vaultAPIClient(vaultConfig.URL, insecureSSLWebhook)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create Vault api client")
	}

	return v.createClient(config, vaultConfig, jwt)
}

func (v *VaultClientFactory) createClient(config *api.Config, vaultConfig vault.Vault, jwt string) (*api.Client, error) {
	vaultClient, err := api.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating vault client")
	}

	// Wait for vault to be ready
	log.Logger().Debugf("Connecting to vault on %s", vaultClient.Address())
	err = waitForVault(vaultClient, healthInitialRetryDelay, healthRetryTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "wait for vault to be initialized and unsealed")
	}

	token, err := getTokenFromVault(vaultConfig.ServiceAccountName, jwt, vaultConfig.KubernetesAuthPath, vaultClient, authRetryTimeout)
	if err != nil {
		return nil, errors.Wrapf(err, "getting Vault authentication token")
	}
	vaultClient.SetToken(token)

	// Wait for KV secret engine V2 to be configured
	err = waitForKVEngine(vaultClient, vaultConfig.SecretEngineMountPoint, kvEngineInitialRetryDelay, kvEngineRetryTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "wait for vault kv engine to be configured")
	}

	return vaultClient, nil
}

// GetConfigData generates the information necessary to configure an api.Client object
// Returns the api.Config object, the JWT needed to create the auth user in vault, and an error if present
func (v *VaultClientFactory) GetConfigData(name string, namespace string, useIngressURL, insecureSSLWebhook bool) (config *api.Config, jwt string, saName string, err error) {
	if namespace == "" {
		namespace = v.defaultNamespace
	}

	vlt, err := v.Selector.GetVault(name, namespace, useIngressURL)
	if err != nil {
		return nil, "", "", err
	}

	if os.Getenv(vault.LocalVaultEnvVar) != "" && !useIngressURL {
		vlt.URL = os.Getenv(vault.LocalVaultEnvVar)
	}

	serviceAccount, err := v.getServiceAccountFromVault(vlt)
	token, err := serviceaccount.GetServiceAccountToken(v.kubeClient, namespace, serviceAccount.Name)
	cfg, err := v.vaultAPIClient(vlt.URL, insecureSSLWebhook)
	if err != nil {
		return nil, "", "", errors.Wrapf(err, "unable to create Vault api client")
	}

	return cfg, token, serviceAccount.Name, err
}

func (v *VaultClientFactory) vaultAPIClient(url string, insecureSSLWebhook bool) (*api.Config, error) {
	cfg := &api.Config{
		Address:    url,
		MaxRetries: maxRetries,
	}

	if insecureSSLWebhook {
		t := api.TLSConfig{Insecure: true}
		err := cfg.ConfigureTLS(&t)
		if err != nil {
			return nil, errors.Wrap(err, "unable to configure tls")
		}
	}

	return cfg, nil
}

func (v *VaultClientFactory) getServiceAccountFromVault(vault *vault.Vault) (*v1.ServiceAccount, error) {
	return v.kubeClient.CoreV1().ServiceAccounts(vault.Namespace).Get(vault.ServiceAccountName, meta_v1.GetOptions{})
}

func waitForVault(vaultClient *api.Client, initialDelay, timeout time.Duration) error {
	return util.RetryWithInitialDelaySlower(initialDelay, timeout, func() error {
		hr, err := vaultClient.Sys().Health()
		if err == nil && hr != nil && hr.Initialized && !hr.Sealed {
			return nil
		}
		log.Logger().Info("Waiting for vault to be initialized and unsealed...")
		if err != nil {
			return errors.Wrap(err, "reading vault health")
		}
		if hr != nil {
			return fmt.Errorf("vault health: initialized=%t, sealed=%t", hr.Initialized, hr.Sealed)
		}
		return errors.New("failed to read vault health")
	})
}

func waitForKVEngine(vaultClient *api.Client, secretEngineMountPoint string, initialDelay, timeout time.Duration) error {
	return util.RetryWithInitialDelaySlower(initialDelay, timeout, func() error {
		if _, err := vaultClient.Logical().Read(fmt.Sprintf("%s/%s", secretEngineMountPoint, kvEngineConfigPath)); err != nil {
			log.Logger().Infof("Waiting for KV secrets engine on %s to be configured...", secretEngineMountPoint)
			return errors.Wrap(err, "checking KV secrets engine config")
		}

		payload := map[string]interface{}{
			"data": map[string]string{
				"test": "write",
			},
		}
		if _, err := vaultClient.Logical().Write(fmt.Sprintf("%s/%s", secretEngineMountPoint, kvEngineWriteCheckPath), payload); err != nil {
			log.Logger().Info("Waiting for KV secrets engine to be ready for write...")
			return errors.Wrap(err, "checking KV secrets engine ready for write")
		}
		return nil
	})
}

func getTokenFromVault(role string, jwt string, kubernetesAuthPath string, vaultClient *api.Client, timeout time.Duration) (string, error) {
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

	clientToken := ""
	err := util.Retry(timeout, func() error {
		sec, err := vaultClient.Logical().Write(fmt.Sprintf("/auth/%s/login", kubernetesAuthPath), m)
		if err == nil {
			clientToken = sec.Auth.ClientToken
			return nil
		}
		return errors.Wrapf(err, "auth with %s login", kubernetesAuthPath)
	})

	return clientToken, err
}
