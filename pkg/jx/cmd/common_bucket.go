package cmd

import (
	"context"
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gocloud.dev/blob"
	"io"
	"net/url"
	"time"
)

// CreateBucketValues contains the values to create a Bucket on cloud storage
type CreateBucketValues struct {
	Bucket     string
	BucketKind string

	// GKE specific values
	GKEProjectID string
	GKEZone      string
}

// addCreateBucketFlags adds the CLI arguments to be able to specify to create a new bucket along with any cloud specific parameters
func (cb *CreateBucketValues) addCreateBucketFlags(cmd *cobra.Command, ) {
	cmd.Flags().StringVarP(&cb.Bucket, "bucket", "", "", "Specify the name of the bucket to use")
	cmd.Flags().StringVarP(&cb.BucketKind, "bucket-kind", "", "", "The kind of bucket to use like 'gs, s3, azure' etc")
	cmd.Flags().StringVarP(&cb.GKEProjectID, "gke-project-id", "", "", "Google Project ID to use for a new bucket")
	cmd.Flags().StringVarP(&cb.GKEZone, "gke-zone", "", "", "The GKE zone (e.g. us-central1-a) where the new bucket will be created")
}


// IsEmpty returns true if there is no bucket name specified
func (cb *CreateBucketValues) IsEmpty() bool {
	return cb.Bucket == ""
}


// createBucket creates a new bucket using the create bucket values and team settings returning the newly created bucket URL
func (o *CommonOptions) createBucket(cb *CreateBucketValues, settings *v1.TeamSettings) (string, error) {
	bucketURL, err := buckets.CreateBucketURL(cb.Bucket, cb.BucketKind, settings)
	if err != nil {
		return bucketURL, errors.Wrapf(err, "failed to create the bucket URL for %s", cb.Bucket)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*20)
	bucket, err := blob.Open(ctx, bucketURL)
	if err != nil {
		return bucketURL, errors.Wrapf(err, "failed to open the bucket for %s", bucketURL)
	}

	// lets check if the bucket exists
	iter := bucket.List(nil)
	obj, err := iter.Next(ctx)
	if err != nil {
		if err == io.EOF {
			log.Infof("bucket %s is empty\n", bucketURL)
		} else {
			log.Infof("The bucket %s does not exist yet so lets create it...\n", util.ColorInfo(bucketURL))
			err = o.createBucketFromURL(bucketURL, bucket, cb)
			if err != nil {
				return bucketURL, errors.Wrapf(err, "failed to create the bucket for %s", bucketURL)
			}
		}
	} else {
		log.Infof("Found item in bucket %s for %s\n", bucketURL, obj.Key)
	}
	return bucketURL, nil
}


// createBucket creates a bucket if it does not already exist
func (o *CommonOptions) createBucketFromURL(bucketURL string, bucket *blob.Bucket, cb *CreateBucketValues) error {
	u, err := url.Parse(bucketURL)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "gs":
		return o.createGcsBucket(u, bucket, cb)
	default:
		return fmt.Errorf("Cannot create a bucket for provider %s", bucketURL)
	}
}

func (o *CommonOptions) createGcsBucket(u *url.URL, bucket *blob.Bucket, cb *CreateBucketValues) error {
	var err error
	if cb.GKEProjectID == "" {
		kubeClient, ns, err := o.KubeClientAndDevNamespace()
		if err != nil {
			return err
		}
		data, err := kube.ReadInstallValues(kubeClient, ns)
		if err != nil {
			log.Warnf("Failed to load install values %s\n", err)
		} else if data != nil {
			cb.GKEProjectID = data[kube.ProjectID]
			if cb.GKEZone == "" {
				cb.GKEZone = data[kube.Zone]
			}
		}
	}
	if cb.GKEProjectID == "" {
		cb.GKEProjectID, err = o.getGoogleProjectId()
		if err != nil {
			return err
		}
	}

	err = o.runCommandVerbose(
		"gcloud", "config", "set", "project", cb.GKEProjectID)
	if err != nil {
		return err
	}

	if cb.GKEZone == "" {
		defaultZone := ""
		if cluster, err := gke.ClusterName(o.Kube()); err == nil && cluster != "" {
			if clusterZone, err := gke.ClusterZone(cluster); err == nil {
				defaultZone = clusterZone
			}
		}

		cb.GKEZone, err = o.getGoogleZoneWithDefault(cb.GKEProjectID, defaultZone)
		if err != nil {
			return err
		}
	}

	bucketName := u.Host
	region := gke.GetRegionFromZone(cb.GKEZone)
	err = gke.CreateBucket(cb.GKEProjectID, bucketName, region)
	if err != nil {
		return errors.Wrapf(err, "creating bucket %s in project %s and region %s", bucketName, cb.GKEProjectID, region)
	}
	return nil
}


