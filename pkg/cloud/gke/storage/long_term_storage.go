package storage

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// EnableLongTermStorage will take the cluster install values and a provided bucket name and use it / create a new one for gs
func EnableLongTermStorage(installValues map[string]string, providedBucketName string) (string, error) {
	if providedBucketName != "" {
		return ensureProvidedBucketExists(installValues, providedBucketName)
	} else {
		log.Logger().Info("No bucket name provided for long term storage, creating a new one")
		return createBucket(createUniqueBucketName(installValues))
	}
}

func ensureProvidedBucketExists(installValues map[string]string, providedBucketName string) (string, error) {
	exists, err := gke.BucketExists(installValues[kube.ProjectID], providedBucketName)
	if err != nil {
		return "", errors.Wrap(err, "checking if the provided bucket exists")
	}
	if exists {
		return fmt.Sprintf("gs://%s", providedBucketName), nil
	}

	bucketURL, err := createBucket(providedBucketName, installValues)
	if err == nil {
		return bucketURL, nil
	}
	log.Logger().Warnf("Attempted to create the bucket %s in the project %s but failed, will now create a "+
		"random bucket", providedBucketName, installValues[kube.ProjectID])

	return createBucket(createUniqueBucketName(installValues))
}

func createUniqueBucketName(installValues map[string]string) (string, map[string]string) {
	uuid4, _ := uuid.NewV4()
	bucketName := fmt.Sprintf("%s-lts-%s", installValues[kube.ClusterName], uuid4.String())
	if len(bucketName) > 60 {
		bucketName = bucketName[:60]
	}
	return bucketName, installValues
}

func createBucket(bucketName string, installValues map[string]string) (string, error) {
	bucketURL := fmt.Sprintf("gs://%s", bucketName)
	infoBucketURL := util.ColorInfo(bucketURL)
	log.Logger().Infof("The bucket %s does not exist so lets create it", infoBucketURL)
	region := gke.GetRegionFromZone(installValues[kube.Zone])
	err := gke.CreateBucket(installValues[kube.ProjectID], bucketName, region)
	gke.AddBucketLabel(bucketName, gke.UserLabel())
	if err != nil {
		return "", errors.Wrapf(err, "there was a problem creating the bucket %s in the GKE Project %s",
			bucketName, installValues[kube.ProjectID])
	}
	return bucketURL, err
}
