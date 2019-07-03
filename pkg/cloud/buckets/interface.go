package buckets

// Provider represents a bucket provider
type Provider interface {
	// CreateNewBucketForCluster creates a new dynamically named bucket
	CreateNewBucketForCluster(clusterName string, bucketKind string) (string, error)
	EnsureBucketIsCreated(bucketURL string) error
}
