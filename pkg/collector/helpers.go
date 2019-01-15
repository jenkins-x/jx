package collector

import (
	"fmt"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
)

// NewCollector creates a new collector from the storage configuration
func NewCollector(storageLocation *jenkinsv1.StorageLocation, settings *jenkinsv1.TeamSettings, gitter gits.Gitter) (Collector, error) {
	classifier := storageLocation.Classifier
	if classifier == "" {
		classifier = "default"
	}
	gitURL := storageLocation.GitURL
	if gitURL != "" {
		return NewGitCollector(gitter, gitURL, storageLocation.GetGitBranch())
	}
	bucket := storageLocation.Bucket
	if bucket != "" {
		bucketKind := storageLocation.BucketKind
		if bucketKind == "" {
			bucketKind = settings.KubeProvider
			if bucketKind == "" {
				return nil, fmt.Errorf("Bucket %s has no associated 'bucketKind' and there is no 'kubeProvider' in the TeamSettings", bucket)
			}
		}
	}
/*	httpURL := storageLocation.HttpURL
	if httpURL != "" {
		return o.collectHttpURL(httpURL)
	}
*/
	return nil, fmt.Errorf("Unsupported storage configuration %#v", storageLocation)
}
