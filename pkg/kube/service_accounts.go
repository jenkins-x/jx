package kube

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	subjectKind                  = "ServiceAccount"
	serviceAccountNameAnnotation = "kubernetes.io/service-account.name"
	tokenDataKey                 = "token"
)

// CreateServiceAccount creates a new service account in the given namespace and returns the service account name
func CreateServiceAccount(kubeClient kubernetes.Interface, namespace string, name string) (*v1.ServiceAccount, error) {
	sa, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
	// If a service account already exists just re-use it
	if err == nil {
		return sa, nil
	}

	sa = &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       subjectKind,
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	sa, err = kubeClient.CoreV1().ServiceAccounts(namespace).Create(sa)
	if err != nil {
		return nil, errors.Wrapf(err, "creating service account '%s'", sa)
	}

	return sa, nil
}

// DeleteServiceAccount deletes a service account
func DeleteServiceAccount(kubeClient kubernetes.Interface, namespace string, name string) error {
	_, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		// Nothing to delete because the service account does not exist
		return nil
	}
	return kubeClient.CoreV1().ServiceAccounts(namespace).Delete(name, &metav1.DeleteOptions{})
}

// GetServiceAccountToken return the token of a service account
func GetServiceAccountToken(kubeClient kubernetes.Interface, namespace string, name string) (string, error) {
	secretList, err := kubeClient.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "listing secrets")
	}
	for _, secret := range secretList.Items {
		annotations := secret.ObjectMeta.Annotations
		for k, v := range annotations {
			if k == serviceAccountNameAnnotation && v == name {
				token, ok := secret.Data[tokenDataKey]
				if !ok {
					return "", errors.New("no token found in service account secret")
				}
				return string(token), nil
			}
		}
	}
	return "", fmt.Errorf("no token found for service account name %s", name)
}
