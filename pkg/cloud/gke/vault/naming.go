package vault

import "fmt"

// BucketName creates a Bucket name for a given vault name and cluster name
func BucketName(vaultName string) string {
	return generateName(vaultName, "bucket")
}

// ServiceAccountName creates a service account name for a given vault and cluster name
func ServiceAccountName(vaultName string) string {
	return generateName(vaultName, "sa")
}

// KeyringName creates a keyring name for a given vault and cluster name
func KeyringName(vaultName string) string {
	return generateName(vaultName, "keyring")
}

// KeyName creates a key name for a given vault and cluster name
func KeyName(vaultName string) string {
	return generateName(vaultName, "key")
}

// GcpServiceAccountSecretName builds the secret name where the GCP service account is stored
func GcpServiceAccountSecretName(vaultName string) string {
	return generateName(vaultName, "gcp-sa")
}

func generateName(vaultName string, name string) string {
	return fmt.Sprintf("%s-%s", vaultName, name)
}
