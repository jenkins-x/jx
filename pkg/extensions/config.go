package extensions

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const ExtensionsConfigDefaultConfigMap = "jenkins-x-extensions"

func GetOrCreateExtensionsConfig(kubeClient kubernetes.Interface, ns string) (*corev1.ConfigMap, error) {
	extensionsConfig, err := kubeClient.CoreV1().ConfigMaps(ns).Get(ExtensionsConfigDefaultConfigMap, metav1.GetOptions{})
	if err != nil {
		// ConfigMap doesn't exist, create it
		extensionsConfig, err = kubeClient.CoreV1().ConfigMaps(ns).Create(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: ExtensionsConfigDefaultConfigMap,
			},
			Data: make(map[string]string),
		})
		if err != nil {
			return nil, err
		}
	}
	return extensionsConfig, nil
}
