package secrets

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const vaultSecretsMarker = "useVaultForSecrets"

// SecretLocation interfaces to identify where is the secrets location
type SecretLocation interface {
	// InVault returns whether secrets are stored in Vault
	InVault() bool
	// SetInVault sets whether secrets are stored in Vault or not
	SetInVault(useVault bool) error
}

type secretLocation struct {
	kubeClient kubernetes.Interface
	namespace  string
	usingVault *bool // use a tri-state boolean. nil means uninitialised (so need to lookup from cluster)
}

// NewSecretLocation creates a SecretLocation
func NewSecretLocation(kubeClient kubernetes.Interface, namespace string) SecretLocation {
	return &secretLocation{
		kubeClient: kubeClient,
		namespace:  namespace,
	}
}

// UsingVaultForSecrets returns true if the cluster has been configured to store secrets in vault
func (s *secretLocation) InVault() bool {
	if s.usingVault == nil {
		configMap, err := getInstallConfigMap(s.kubeClient, s.namespace)
		b := false
		if err == nil {
			b = configMap[vaultSecretsMarker] != ""
		}
		s.usingVault = &b
	}
	return *s.usingVault
}

// UseVaultForSecrets configures the cluster's installation config map to denote that secrets should be stored in vault
func (s *secretLocation) SetInVault(useVault bool) error {
	_, err := kube.DefaultModifyConfigMap(s.kubeClient, s.namespace, kube.ConfigMapNameJXInstallConfig, func(configMap *v1.ConfigMap) error {
		if useVault {
			configMap.Data[vaultSecretsMarker] = "true"
		} else {
			delete(configMap.Data, vaultSecretsMarker)
		}
		s.usingVault = &useVault
		return nil
	}, nil)
	if err != nil {
		return errors.Wrapf(err, "saving vault flag in configmap %s", kube.ConfigMapNameJXInstallConfig)
	}
	return nil
}

func getInstallConfigMap(kubeClient kubernetes.Interface, namespace string) (map[string]string, error) {
	configMap, err := kube.GetConfigMapData(kubeClient, kube.ConfigMapNameJXInstallConfig, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "getting configmap %s", kube.ConfigMapNameJXInstallConfig)
	}
	return configMap, nil
}
