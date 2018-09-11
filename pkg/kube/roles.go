package kube

import (
	"sort"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetTeamRoles returns the roles for the given team dev namespace
func GetTeamRoles(kubeClient kubernetes.Interface, ns string) (map[string]*rbacv1.Role, []string, error) {
	m := map[string]*rbacv1.Role{}

	names := []string{}
	resources, err := kubeClient.RbacV1().Roles(ns).List(metav1.ListOptions{
		LabelSelector: LabelKind + "=" + ValueKindEnvironmentRole,
	})
	if err != nil {
		return m, names, err
	}
	for _, env := range resources.Items {
		n := env.Name
		copy := env
		m[n] = &copy
		if n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return m, names, nil
}
