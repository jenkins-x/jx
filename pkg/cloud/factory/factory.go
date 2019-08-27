package factory

import (
	"github.com/jenkins-x/jx/pkg/cloud"
	amazonStorage "github.com/jenkins-x/jx/pkg/cloud/amazon/storage"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/cloud/gke/storage"
	"github.com/jenkins-x/jx/pkg/config"
)

// NewBucketProvider creates a new bucket provider for a given Kubernetes provider
func NewBucketProvider(requirements *config.RequirementsConfig) buckets.Provider {
	switch requirements.Cluster.Provider {
	case cloud.GKE:
		return storage.NewGKEBucketProvider(requirements)
	case cloud.EKS:
	case cloud.AWS:
		return amazonStorage.NewAmazonBucketProvider(requirements)
	default:
		return nil
	}
	return nil
}
