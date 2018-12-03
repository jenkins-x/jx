package vault

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/kube/serviceaccount"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	gkeServiceAccountSecretKey = "service-account.json"
)

var (
	ServiceAccountRoles = []string{"roles/storage.objectAdmin",
		"roles/cloudkms.admin",
		"roles/cloudkms.cryptoKeyEncrypterDecrypter",
	}
)

// KmsConfig keeps the configuration for Google KMS service
type KmsConfig struct {
	Keyring  string
	Key      string
	Location string
	project  string
}

// This is a loose collection of methods needed to set up a vault in GKE.
// If they are generic enough and needed elsewhere, we can move them up one level to more generic GCP methods.

// CreateKmsConfig creates a KMS config for the GKE Vault
func CreateKmsConfig(vaultName, clusterName, projectId string) (*KmsConfig, error) {
	config := &KmsConfig{
		Keyring:  KeyringName(vaultName, clusterName),
		Key:      KeyName(vaultName, clusterName),
		Location: gke.KmsLocation,
		project:  projectId,
	}

	err := gke.CreateKmsKeyring(config.Keyring, config.project)
	if err != nil {
		return nil, errors.Wrapf(err, "creating kms keyring '%s'", config.Keyring)
	}

	err = gke.CreateKmsKey(config.Key, config.Keyring, config.project)
	if err != nil {
		return nil, errors.Wrapf(err, "crating the kms key '%s'", config.Key)
	}
	return config, nil
}

// CreateGCPServiceAccount creates a service account in GCP for the vault service
func CreateGCPServiceAccount(kubeClient kubernetes.Interface, vaultName, namespace, clusterName, projectId string) (string, error) {
	serviceAccountDir, err := ioutil.TempDir("", "gke")
	if err != nil {
		return "", errors.Wrap(err, "creating a temporary folder where the service account will be stored")
	}
	defer os.RemoveAll(serviceAccountDir)

	serviceAccountName := ServiceAccountName(vaultName, clusterName)
	if err != nil {
		return "", err
	}
	serviceAccountPath, err := gke.GetOrCreateServiceAccount(serviceAccountName, projectId, serviceAccountDir, ServiceAccountRoles)
	if err != nil {
		return "", errors.Wrap(err, "creating the service account")
	}

	secretName, err := storeGCPServiceAccountIntoSecret(kubeClient, serviceAccountPath, vaultName, namespace, clusterName)
	if err != nil {
		return "", errors.Wrap(err, "storing the service account into a secret")
	}
	return secretName, nil
}

// CreateBucket Creates a bucket in GKE to store the backend (encrypted) data for vault
func CreateBucket(vaultName, clusterName, projectId, zone string) (string, error) {
	bucketName := BucketName(vaultName, clusterName)
	exists, err := gke.BucketExists(projectId, bucketName)
	if err != nil {
		return "", errors.Wrap(err, "checking if Vault GCS bucket exists")
	}
	if exists {
		return bucketName, nil
	}

	if zone == "" {
		return "", errors.New("GKE zone must be provided in 'gke-zone' option")
	}
	region := gke.GetRegionFromZone(zone)
	err = gke.CreateBucket(projectId, bucketName, region)
	if err != nil {
		return "", errors.Wrap(err, "creating Vault GCS bucket")
	}
	return bucketName, nil
}

// CreateAuthServiceAccount creates a Serivce Account for the Auth service for vault
func CreateAuthServiceAccount(client kubernetes.Interface, vaultName, namespace, clusterName string) (string, error) {
	serviceAccountName := AuthServiceAccountName(vaultName, clusterName)
	_, err := serviceaccount.CreateServiceAccount(client, namespace, serviceAccountName)
	if err != nil {
		return "", errors.Wrap(err, "creating vault auth service account")
	}
	return serviceAccountName, nil
}

// VaultGcpServiceAccountSecretName builds the secret name where the GCP service account is stored
func VaultGcpServiceAccountSecretName(vaultName string, clusterName string) string {
	return fmt.Sprintf("%s-%s-gcp-sa", clusterName, vaultName)
}

func storeGCPServiceAccountIntoSecret(client kubernetes.Interface, serviceAccountPath, vaultName, namespace, clusterName string) (string, error) {
	serviceAccount, err := ioutil.ReadFile(serviceAccountPath)
	if err != nil {
		return "", errors.Wrapf(err, "reading the service account from file '%s'", serviceAccountPath)
	}

	secretName := VaultGcpServiceAccountSecretName(vaultName, clusterName)
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Data: map[string][]byte{
			gkeServiceAccountSecretKey: serviceAccount,
		},
	}

	secrets := client.CoreV1().Secrets(namespace)
	_, err = secrets.Get(secretName, metav1.GetOptions{})
	if err != nil {
		_, err = secrets.Create(secret)
	} else {
		_, err = secrets.Update(secret)
	}
	return secretName, nil
}
