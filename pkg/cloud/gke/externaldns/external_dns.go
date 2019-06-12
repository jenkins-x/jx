package externaldns

import (
	"github.com/jenkins-x/jx/pkg/cloud/gke"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	serviceAccountSecretKey = "credentials.json"
	DefaultExternalDNSAbbreviation = "dn"
)

var (
	serviceAccountRoles = []string{
		"roles/dns.admin",
	}
)

// CreateExternalDNSGCPServiceAccount creates a service account in GCP for ExternalDNS
func CreateExternalDNSGCPServiceAccount(kubeClient kubernetes.Interface, externalDNSName, namespace, clusterName, projectID string) (string, error) {

	gcpServiceAccountSecretName, err := gke.CreateGCPServiceAccount(kubeClient, externalDNSName, DefaultExternalDNSAbbreviation, namespace, clusterName, projectID, serviceAccountRoles, serviceAccountSecretKey)
	if err != nil {
		return "", errors.Wrap(err, "creating the ExternalDNS GCP Service Account")
	}
	return gcpServiceAccountSecretName, nil
}
