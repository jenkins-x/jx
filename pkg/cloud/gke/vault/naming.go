package vault

import "fmt"

// BucketName creates a Bucket name for a given vault name and cluster name
func BucketName(vaultName string, clusterName string) string {
	return generatePrefix(vaultName, clusterName) + "bucket"
}

// ServiceAccountName creates a service account name for a given vault and cluster name
func ServiceAccountName(vaultName string, clusterName string) string {
	return generatePrefix(vaultName, clusterName) + "sa"
}

// AuthServiceAccountName creates a service account name for a given vault and cluster name
func AuthServiceAccountName(vaultName string, clusterName string) string {
	return generatePrefix(vaultName, clusterName) + "auth-sa"
}

// KeyringName creates a keyring name for a given vault and cluster name
func KeyringName(vaultName string, clusterName string) string {
	return generatePrefix(vaultName, clusterName) + "keyring"
}

// KeyName creates a key name for a given vault and cluster name
func KeyName(vaultName string, clusterName string) string {
	return generatePrefix(vaultName, clusterName) + "key"
}

func generatePrefix(vaultName string, clusterName string) string {
	return fmt.Sprintf("%s-%s-", clusterName, vaultName)
}
