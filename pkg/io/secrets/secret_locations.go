package secrets

import (
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const vaultSecretsMarker = "useVaultForSecrets"

// UseVaultForSecrets configures the cluster's installation config map to denote that secrets should be stored in vault
func UseVaultForSecrets(kubeClient kubernetes.Interface, namespace string, useVault bool) {
	_, err := kube.DefaultModifyConfigMap(kubeClient, namespace, kube.ConfigMapNameJXInstallConfig, func(configMap *v1.ConfigMap) error {
		if useVault {
			configMap.Data[vaultSecretsMarker] = "true"
		} else {
			delete(configMap.Data, vaultSecretsMarker)
		}
		return nil
	}, nil)
	if err != nil {
		logrus.Errorf("Error saving configmap %s: %v", kube.ConfigMapNameJXInstallConfig, err)
	}
}

// UsingVaultForSecrets returns true if the cluster has been configured to store secrets in vault
func UsingVaultForSecrets(kubeClient kubernetes.Interface, namespace string) bool {
	configMap := getInstallConfigMap(kubeClient, namespace)
	return configMap[vaultSecretsMarker] != ""
}

func getInstallConfigMap(kubeClient kubernetes.Interface, namespace string) map[string]string {
	configMap, err := kube.GetConfigMapData(kubeClient, kube.ConfigMapNameJXInstallConfig, namespace)
	if err != nil {
		logrus.Errorf("Error getting configmap %s: %v", kube.ConfigMapNameJXInstallConfig, err)
	}
	return configMap
}
