package amazon

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// CreateS3Bucket creates a new S3 bucket in the default region with the given bucket name
// returning the location string
func CreateS3Bucket(bucketName string, region string) (string, error) {
	location := ""
	sess, defaultRegion, err := NewAwsSession()
	if err != nil {
		return location, err
	}
	if region == "" {
		region = defaultRegion
	}

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(region),
		},
	}
	svc := s3.New(sess)
	result, err := svc.CreateBucket(input)
	if result != nil && result.Location != nil {
		location = *result.Location
	}
	return location, err
}
