package kube

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sort"
)

// GetSecrets returns a map of the Secrets along with a sorted list of names
func GetSecrets(kubeClient kubernetes.Interface, ns string) (map[string]*v1.Secret, []string, error) {
	m := map[string]*v1.Secret{}

	names := []string{}
	resourceList, err := kubeClient.CoreV1().Secrets(ns).List(metav1.ListOptions{})
	if err != nil {
		return m, names, err
	}
	for _, resource := range resourceList.Items {
		n := resource.Name
		copy := resource
		m[n] = &copy
		if n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return m, names, nil
}
