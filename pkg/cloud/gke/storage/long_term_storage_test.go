package storage

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestCreateAValidBucketNameWithLongClusterNameAndDashAtCharacterSixty(t *testing.T) {
	bucketName := createUniqueBucketNameForCluster("rrehhhhhhhhhhhhhhhhhhhhhhhhhj3j3k2kwkdkjdbiwabduwabduoawbdb-dbwdbwaoud")
	assert.NotNil(t, bucketName, "it should always generate a name")
	assert.False(t, strings.HasSuffix(bucketName, "-"), "the bucket can't end with a dash")
}

func TestCreateAValidBucketNameWithSmallClusterName(t *testing.T) {
	bucketName := createUniqueBucketNameForCluster("cluster")
	assert.NotNil(t, bucketName, "it should always generate a name")
	assert.False(t, strings.HasSuffix(bucketName, "-"), "the bucket can't end with a dash")
}
