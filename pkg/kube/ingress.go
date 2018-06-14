package kube

import (
	"fmt"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetIngress(client kubernetes.Interface, ns, name string) (string, error) {

	ing, err := client.ExtensionsV1beta1().Ingresses(ns).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get ingress rule %s. error: %v", name, err)
	}
	if ing == nil {
		return "", fmt.Errorf("failed to find ingress rule %s", name)
	}

	// default to the first rule
	if len(ing.Spec.Rules) > 0 {
		return ing.Spec.Rules[0].Host, nil
	}
	return "", fmt.Errorf("no hostname found for ingress rule %s", name)
}
