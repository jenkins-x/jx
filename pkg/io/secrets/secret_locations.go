package secrets

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const vaultSecretsMarker = "useVaultForSecrets"

type SecretLocation interface {
	// InVault returns whether secrets are stored in Vault
	InVault() bool
	// SetInVault sets whether secrets are stored in Vault or not
	SetInVault(useVault bool)
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
		configMap := getInstallConfigMap(s.kubeClient, s.namespace)
		b := configMap[vaultSecretsMarker] != ""
		s.usingVault = &b
	}
	return *s.usingVault
}

// UseVaultForSecrets configures the cluster's installation config map to denote that secrets should be stored in vault
func (s *secretLocation) SetInVault(useVault bool) {
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
		logrus.Errorf("Error saving configmap %s: %v", kube.ConfigMapNameJXInstallConfig, err)
	}
}

func getInstallConfigMap(kubeClient kubernetes.Interface, namespace string) map[string]string {
	configMap, err := kube.GetConfigMapData(kubeClient, kube.ConfigMapNameJXInstallConfig, namespace)
	if err != nil {
		logrus.Errorf("Error getting configmap %s: %v", kube.ConfigMapNameJXInstallConfig, err)
	}
	return configMap
}
