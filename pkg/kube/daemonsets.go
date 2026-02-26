package kube

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//IsDaemonSetExists checks if a daemonset exists in the namespace
func IsDaemonSetExists(client kubernetes.Interface, name, namespace string) (bool, error) {
	options := metav1.GetOptions{}

	_, err := client.AppsV1().DaemonSets(namespace).Get(name, options)
	if err != nil {
		return false, err
	}
	return true, nil
}
