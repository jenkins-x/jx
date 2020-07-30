// +build unit

package storage

import (
	"strings"
	"testing"

	aks_test "github.com/jenkins-x/jx/v2/pkg/cloud/aks/mocks"
	"github.com/jenkins-x/jx/v2/pkg/config"
	"github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
)

func TestCreateNewBucketForClusterWithLongClusterNameAndDashAtCharacterSixty(t *testing.T) {
	p := AKSBucketProvider{
		AzureStorage: aks_test.NewMockAzureStorage(),
		Requirements: &config.RequirementsConfig{
			Velero: config.VeleroConfig{
				ServiceAccount: "teststorage",
			},
		},
	}

	pegomock.When(p.AzureStorage.ContainerExists(pegomock.AnyString())).ThenReturn(true, nil)

	bucketName, err := p.CreateNewBucketForCluster("rrehhhhhhhhhhhhhhhhhhhhhhhhhj3j3k2kwkdkjdbiwabduwabduoawbdb-dbwdbwaoud", "logs")
	assert.NoError(t, err)
	assert.NotNil(t, bucketName, "it should always generate a name")
	assert.False(t, strings.HasSuffix(bucketName, "-"), "the bucket can't end with a dash")
	assert.Equal(t, "https://teststorage.blob.core.windows.net/rrehhhhhhhhhhhhhhhhhhhhhhhhhj3j3k2kwkdkjdbiwabduwabduoawbdb-dbw", bucketName)
}

func TestCreateNewBucketForClusterWithSmallClusterName(t *testing.T) {
	p := AKSBucketProvider{
		AzureStorage: aks_test.NewMockAzureStorage(),
		Requirements: &config.RequirementsConfig{
			Velero: config.VeleroConfig{
				ServiceAccount: "teststorage",
			},
		},
	}

	pegomock.When(p.AzureStorage.ContainerExists(pegomock.AnyString())).ThenReturn(true, nil)

	bucketName, err := p.CreateNewBucketForCluster("cluster", "logs")
	assert.NoError(t, err)
	assert.NotNil(t, bucketName, "it should always generate a name")
	assert.False(t, strings.HasSuffix(bucketName, "-"), "the bucket can't end with a dash")
}
