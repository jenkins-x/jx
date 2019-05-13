package storage

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

func EnableLongTermStorage(installValues map[string]string, providedBucketName string,
	createBucketFN func(bucketName string, bucketKind string) (string, error)) (string, error) {
	var bucketName string
	if providedBucketName != "" {
		exists, err := gke.BucketExists(installValues[kube.ProjectID], providedBucketName)
		if err != nil {
			return "", errors.Wrap(err, "checking if the provided bucket exists")
		}
		if exists {
			bucketName = providedBucketName
			return fmt.Sprintf("gs://%s", bucketName), nil
		}

		bucketURL, err := createBucketFN(providedBucketName, "gs")
		if err == nil {
			return bucketURL, nil
		}
		log.Warnf("Attempted to create the bucket %s in the project %s but failed, will now create a "+
			"random bucket", providedBucketName, installValues[kube.ProjectID])
	}

	if providedBucketName == "" {
		log.Info("No bucket name provided for long term storage, creating a new one")
	}

	uuid4, _ := uuid.NewV4()
	bucketName = fmt.Sprintf("%s-lts-%s", installValues[kube.ClusterName], uuid4.String())
	if len(bucketName) > 60 {
		bucketName = bucketName[:60]
	}

	return createBucketFN(bucketName, "gs")
}
