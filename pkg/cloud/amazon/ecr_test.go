package amazon_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/stretchr/testify/assert"
)

func TestCreateNewSessionWithDefaultRegion(t *testing.T) {
	// TODO Refactor for encapsulation
	os.Setenv("AWS_REGION", "")
	os.Setenv("AWS_DEFAULT_REGION", "")
	_, region, err := amazon.NewAwsSession()
	assert.Nil(t, err)
	assert.Equal(t, "us-west-2", region)
}

func TestCreateNewSessionWithRegionFromAwsRegion(t *testing.T) {
	// TODO Refactor for encapsulation
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_DEFAULT_REGION", "")
	_, region, err := amazon.NewAwsSession()
	assert.Nil(t, err)
	assert.Equal(t, "us-east-1", region)
}
