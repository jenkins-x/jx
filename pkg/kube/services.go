package kube

import (
	"k8s.io/client-go/kubernetes"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ExposeURLAnnotation = "fabric8.io/exposeUrl"
)

func FindServiceURL(client *kubernetes.Clientset, namespace string, name string) (string, error) {
	options := meta_v1.GetOptions{}
	svc, err := client.CoreV1().Services(namespace).Get(name, options)
	if err != nil {
		return "", err
	}
	url := ""
	if svc.Annotations != nil {
		url = svc.Annotations[ExposeURLAnnotation]
	}
	return url, nil
}
