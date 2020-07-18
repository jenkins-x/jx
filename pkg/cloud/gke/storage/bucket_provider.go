package storage

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	uuid2 "github.com/google/uuid"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/v2/pkg/cloud/gke"
	"github.com/jenkins-x/jx/v2/pkg/config"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
)

var (
	defaultBucketWriteTimeout = 20 * time.Second
)

// GKEBucketProvider the bucket provider for GKE
type GKEBucketProvider struct {
	Requirements *config.RequirementsConfig
	gcloud       gke.GClouder
}

// CreateNewBucketForCluster creates a new dynamic bucket
func (b *GKEBucketProvider) CreateNewBucketForCluster(clusterName string, bucketKind string) (string, error) {
	uuid := uuid2.New()
	bucketURL := fmt.Sprintf("gs://%s-%s-%s", clusterName, bucketKind, uuid.String())
	if len(bucketURL) > 60 {
		bucketURL = bucketURL[:60]
	}
	if strings.HasSuffix(bucketURL, "-") {
		bucketURL = bucketURL[:59]
	}
	err := b.EnsureBucketIsCreated(bucketURL)
	if err != nil {
		return bucketURL, errors.Wrapf(err, "failed to create bucket %s", bucketURL)
	}

	return bucketURL, nil
}

// EnsureBucketIsCreated ensures the bucket URL is createtd
func (b *GKEBucketProvider) EnsureBucketIsCreated(bucketURL string) error {
	project := b.Requirements.Cluster.ProjectID
	if project == "" {
		return fmt.Errorf("requirements do not specify a project")
	}
	u, err := url.Parse(bucketURL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse bucket name from %s", bucketURL)
	}
	bucketName := u.Host

	exists, err := b.gcloud.BucketExists(project, bucketName)
	if err != nil {
		return errors.Wrap(err, "checking if the provided bucket exists")
	}
	if exists {
		return nil
	}

	infoBucketURL := util.ColorInfo(bucketURL)
	log.Logger().Infof("The bucket %s does not exist so lets create it", infoBucketURL)
	region := gke.GetRegionFromZone(b.Requirements.Cluster.Zone)
	err = b.gcloud.CreateBucket(project, bucketName, region)
	b.gcloud.AddBucketLabel(bucketName, b.gcloud.UserLabel())
	if err != nil {
		return errors.Wrapf(err, "there was a problem creating the bucket %s in the GKE Project %s",
			bucketName, project)
	}
	return nil
}

// UploadFileToBucket uploads a file to the provided GCS bucket with the provided outputName
func (b *GKEBucketProvider) UploadFileToBucket(reader io.Reader, key string, bucketURL string) (string, error) {
	log.Logger().Debugf("Uploading to bucket %s with key %s", bucketURL, key)
	err := buckets.WriteBucket(bucketURL, key, reader, defaultBucketWriteTimeout)
	return bucketURL + "/" + key, err
}

// DownloadFileFromBucket downloads a file from GCS from the given bucketURL and server its contents with a bufio.Scanner
func (b *GKEBucketProvider) DownloadFileFromBucket(bucketURL string) (io.ReadCloser, error) {
	return gke.StreamTransferFileFromBucket(bucketURL)
}

// NewGKEBucketProvider create a new provider for GKE
func NewGKEBucketProvider(requirements *config.RequirementsConfig) buckets.Provider {
	return &GKEBucketProvider{
		Requirements: requirements,
		gcloud:       &gke.GCloud{},
	}
}
