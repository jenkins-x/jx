package kube

import (
	"sort"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

// DefaultModifySecret default implementation of a function to modify
func DefaultModifySecret(kubeClient kubernetes.Interface, ns string, name string, fn func(env *v1.Secret) error, defaultSecret *v1.Secret) (*v1.Secret, error) {
	secretInterface := kubeClient.CoreV1().Secrets(ns)

	create := false
	secret, err := secretInterface.Get(name, metav1.GetOptions{})
	if err != nil {
		create = true
		initialSecret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			Data: map[string][]byte{},
		}
		if defaultSecret != nil {
			initialSecret = *defaultSecret
		}
		secret = &initialSecret
	}
	err = fn(secret)
	if err != nil {
		return secret, err
	}
	if create {
		log.Infof("Creating Secret %s in namespace %s\n", util.ColorInfo(name), util.ColorInfo(ns))
		_, err = secretInterface.Create(secret)
		if err != nil {
			return secret, errors.Wrapf(err, "Failed to create Secret %s in namespace %s", name, ns)
		}
		return secret, err
	}
	log.Infof("Updating Secret %s in namespace %s\n", util.ColorInfo(name), util.ColorInfo(ns))
	_, err = secretInterface.Update(secret)
	if err != nil {
		return secret, errors.Wrapf(err, "Failed to update Secret %s in namespace %s", name, ns)
	}
	return secret, nil
}
