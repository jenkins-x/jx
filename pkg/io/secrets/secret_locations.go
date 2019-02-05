package secrets

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// SecretsLocationKind key in the config map which stored the location where the secrets are stored
	SecretsLocationKey = "secretsLocation"
)

// SecretsLocationKind type for secrets location kind
type SecretsLocationKind string

const (
	// FileSystemLocationKind indicates that secrets location is the file system
	FileSystemLocationKind SecretsLocationKind = "fileSystem"
	// VaultLocationKind indicates that secrets location is vault
	VaultLocationKind SecretsLocationKind = "vault"
	// KubeLocationKind inidcates that secrets location is in Kuberntes
	KubeLocationKind SecretsLocationKind = "kube"
)

// SecretLocation interfaces to identify where is the secrets location
type SecretLocation interface {
	// Location returns the location where the secrets are stored
	Location() SecretsLocationKind
	// SecretLocation configure the secrets location
	SetLocation(location SecretsLocationKind) error
}

type secretLocation struct {
	kubeClient kubernetes.Interface
	namespace  string
}

// NewSecretLocation creates a SecretLocation
func NewSecretLocation(kubeClient kubernetes.Interface, namespace string) SecretLocation {
	return &secretLocation{
		kubeClient: kubeClient,
		namespace:  namespace,
	}
}

// Location returns the location of the secrets
func (s *secretLocation) Location() SecretsLocationKind {
	configMap, err := getInstallConfigMapData(s.kubeClient, s.namespace)
	if err != nil {
		return FileSystemLocationKind
	}
	value, ok := configMap[SecretsLocationKey]
	if ok && value == string(VaultLocationKind) {
		return VaultLocationKind
	}
	return FileSystemLocationKind
}

// SetLocation configures the cluster's installation config map to denote that secrets should be stored in vault
func (s *secretLocation) SetLocation(location SecretsLocationKind) error {
	_, err := kube.DefaultModifyConfigMap(s.kubeClient, s.namespace, kube.ConfigMapNameJXInstallConfig, func(configMap *v1.ConfigMap) error {
		configMap.Data[SecretsLocationKey] = string(location)
		return nil
	}, nil)
	if err != nil {
		return errors.Wrapf(err, "saving secrets location in configmap %s", kube.ConfigMapNameJXInstallConfig)
	}
	return nil
}

func getInstallConfigMapData(kubeClient kubernetes.Interface, namespace string) (map[string]string, error) {
	configMap, err := kube.GetConfigMapData(kubeClient, kube.ConfigMapNameJXInstallConfig, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "getting configmap %s", kube.ConfigMapNameJXInstallConfig)
	}
	return configMap, nil
}
