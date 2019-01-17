package kube

import (
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// JXInstallConfig is the struct used to create the jx-install-config configmap
type JXInstallConfig struct {
	Server string `structs:"server" yaml:"server" json:"server"`
	CA     []byte `structs:"ca.crt" yaml:"ca.crt" json:"ca.crt"`
}

func RememberRegion(kubeClient kubernetes.Interface, namespace string, region string) error {
	_, err := DefaultModifyConfigMap(kubeClient, namespace, ConfigMapNameJXInstallConfig, func(configMap *v1.ConfigMap) error {
		configMap.Data[Region] = region
		return nil
	}, nil)
	if err != nil {
		return errors.Wrapf(err, "saving AWS region in ConfigMap %s", ConfigMapNameJXInstallConfig)
	} else {
		return nil
	}
}

func ReadRegion(kubeClient kubernetes.Interface, namespace string) (string, error) {
	data, err := GetConfigMapData(kubeClient, ConfigMapNameJXInstallConfig, namespace)
	if err != nil {
		return "", err
	}
	return data[Region], nil
}