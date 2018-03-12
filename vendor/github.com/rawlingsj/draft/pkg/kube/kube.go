package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

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
