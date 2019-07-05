package factory

import (
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/cloud/gke/storage"
	"github.com/jenkins-x/jx/pkg/config"
)

// NewBucketProvider creates a new provider for kubeernetes provider
func NewBucketProvider(requirements *config.RequirementsConfig) buckets.Provider {
	switch requirements.Cluster.Provider {
	case cloud.GKE:
		return storage.NewGKEBucketProvider(requirements)
	default:
		return nil
	}
}
