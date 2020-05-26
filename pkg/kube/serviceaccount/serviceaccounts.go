package serviceaccount

import (
	"encoding/json"
	"fmt"

	"github.com/jenkins-x/jx/v2/pkg/kube"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type JsonPatch struct {
	ImagePullSecret *[]ImagePullSecret `json:"imagePullSecrets"`
}

type ImagePullSecret struct {
	Name string `json:"name"`
}

// PatchImagePullSecrets patches the specified ImagePullSecrets to the given service account
func PatchImagePullSecrets(kubeClient kubernetes.Interface, ns string, sa string, imagePullSecrets []string) error {
	var ips []ImagePullSecret
	for _, secret := range imagePullSecrets {
		jsonSecret := ImagePullSecret{
			Name: secret,
		}
		ips = append(ips, jsonSecret)
	}
	payload := JsonPatch{
		ImagePullSecret: &ips,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	//log.Logger().Infof("Resultant JSON: %s\n", string(b))
	_, err = kubeClient.CoreV1().ServiceAccounts(ns).Patch(sa, types.StrategicMergePatchType, b)
	if err != nil {
		return err
	}
	return nil
}

const (
	subjectKind  = "ServiceAccount"
	tokenDataKey = "token"
	certDataKey  = "ca.crt"
)

// CreateServiceAccount creates a new services account in the given namespace and returns the service account name
func CreateServiceAccount(kubeClient kubernetes.Interface, namespace string, name string) (*v1.ServiceAccount, error) {
	err := kube.EnsureNamespaceCreated(kubeClient, namespace, nil, nil)
	if err != nil {
		return nil, err
	}
	sa, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
	// If a services account already exists just re-use it
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
func GetServiceAccountToken(kubeClient kubernetes.Interface, namespace string, serviceAccountName string) (string, error) {
	return getServiceAccountSecret(kubeClient, namespace, serviceAccountName, tokenDataKey)
}

// GetServiceAccountCert returns the certificate data for the specified service account in the given namespace.
// Returns an error if an error occurs retrieving the certificate.
func GetServiceAccountCert(kubeClient kubernetes.Interface, namespace string, serviceAccountName string) (string, error) {
	return getServiceAccountSecret(kubeClient, namespace, serviceAccountName, certDataKey)
}

func getServiceAccountSecret(kubeClient kubernetes.Interface, namespace string, serviceAccountName string, key string) (string, error) {
	secretList, err := kubeClient.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "listing secrets")
	}
	for _, secret := range secretList.Items {
		annotations := secret.ObjectMeta.Annotations
		for k, v := range annotations {
			if k == v1.ServiceAccountNameKey && v == serviceAccountName {
				data, ok := secret.Data[key]
				if !ok {
					return "", fmt.Errorf("no %s found for service account %s in namespace %s", key, serviceAccountName, namespace)
				}
				return string(data), nil
			}
		}
	}
	return "", fmt.Errorf("no %s found for service account %s in namespace %s", key, serviceAccountName, namespace)
}
