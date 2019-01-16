package collector

import (
	"context"
	"fmt"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/pkg/errors"
	"gocloud.dev/blob"

	// lets import all the blob providers we need
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
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
	ctx := context.Background()
	u := storageLocation.BucketURL
	if u == "" {
		return nil, fmt.Errorf("No GitURL or BucketURL is configured for the storage location in the TeamSettings")
	}
	bucket, err := blob.Open(ctx, u)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open bucket %s", u)
	}
	return NewBucketCollector(u, bucket, classifier)
}
