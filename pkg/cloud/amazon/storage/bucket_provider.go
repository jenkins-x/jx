package storage

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// AmazonBucketProvider the bucket provider for AWS
type AmazonBucketProvider struct {
	Requirements *config.RequirementsConfig
	api          s3iface.S3API
}

func (b *AmazonBucketProvider) s3() (s3iface.S3API, error) {
	if b.api != nil {
		return b.api, nil
	}
	region := b.Requirements.Cluster.Region
	if region == "" {
		return nil, errors.New("requirements do not specify a cluster region")
	}
	sess, err := amazon.NewAwsSession("", region)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AWS session")
	}
	b.api = s3.New(sess)

	return b.api, nil
}

// CreateNewBucketForCluster creates a new dynamic bucket
func (b *AmazonBucketProvider) CreateNewBucketForCluster(clusterName string, bucketKind string) (string, error) {
	uuid4, _ := uuid.NewV4()
	bucketName := fmt.Sprintf("%s-%s-%s", clusterName, bucketKind, uuid4.String())

	// Max length is 63, https://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
	if len(bucketName) > 63 {
		bucketName = bucketName[:63]
	}
	bucketName = strings.TrimRight(bucketName, "-")
	bucketURL := "s3://" + bucketName
	err := b.EnsureBucketIsCreated(bucketURL)
	if err != nil {
		return bucketURL, errors.Wrapf(err, "failed to create bucket %s", bucketURL)
	}

	return bucketURL, nil
}

// EnsureBucketIsCreated ensures the bucket URL is created
func (b *AmazonBucketProvider) EnsureBucketIsCreated(bucketURL string) error {
	svc, err := b.s3()
	if err != nil {
		return err
	}

	u, err := url.Parse(bucketURL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse bucket name from %s", bucketURL)
	}
	bucketName := u.Host

	// Check if bucket exists already
	_, err = svc.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(bucketName)})
	if err == nil {
		return nil // bucket already exists
	}
	aerr, ok := err.(awserr.Error)
	if !ok || aerr.Code() != s3.ErrCodeNoSuchBucket {
		return errors.Wrapf(err, "failed to check if %s bucket exists already", bucketName)
	}

	infoBucketURL := util.ColorInfo(bucketURL)
	log.Logger().Infof("The bucket %s does not exist so lets create it", infoBucketURL)
	_, err = svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(b.Requirements.Cluster.Region),
		},
	})
	if err != nil {
		return errors.Wrapf(err, "there was a problem creating the bucket %s in the AWS", bucketName)
	}
	return nil
}

// NewAmazonBucketProvider create a new provider for AWS
func NewAmazonBucketProvider(requirements *config.RequirementsConfig) buckets.Provider {
	return &AmazonBucketProvider{
		Requirements: requirements,
	}
}
