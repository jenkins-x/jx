package external_dns

import (
	"github.com/jenkins-x/jx/pkg/cloud/gke"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	serviceAccountSecretKey = "credentials.json"
)

var (
	ServiceAccountRoles = []string{
		"roles/dns.admin",
	}
)

// CreateGCPServiceAccount creates a service account in GCP for ExternalDNS
func CreateExternalDNSGCPServiceAccount(kubeClient kubernetes.Interface, externalDNSName, namespace, clusterName, projectId string) (string, error) {

	gcpServiceAccountSecretName, error := gke.CreateGCPServiceAccount(kubeClient, externalDNSName, namespace, clusterName, projectId, ServiceAccountRoles, serviceAccountSecretKey)
	if error != nil {
		return "", errors.Wrap(error, "creating the ExternalDNS GCP Service Account")
	}
	return gcpServiceAccountSecretName, nil
}
