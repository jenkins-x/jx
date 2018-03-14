package kube

import (
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetInClusterClient returns an in cluster Kubernetes client
func GetInClusterClient() (kubernetes.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return k8s, nil
}

// GetOutOfClusterClient returns a client side Kubernetes client
func GetOutOfClusterClient() (kubernetes.Interface, error) {
	k8scfg := os.Getenv("KUBECONFIG")
	if k8scfg == "" {
		k8scfg = os.Getenv("HOME") + "/.kube/config"
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", k8scfg)
	if err != nil {
		return nil, err
	}
	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return k8s, nil
}
