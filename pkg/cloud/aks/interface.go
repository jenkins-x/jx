package aks

// AzureStorage Interface for Azure Storage commands
type AzureStorage interface {
	ContainerExists(bucketURL string) (bool, error)
	CreateContainer(bucketURL string) error
	GetStorageAccessKey(storageAccount string) (string, error)
}
