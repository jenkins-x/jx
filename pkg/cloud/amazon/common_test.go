package amazon_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/stretchr/testify/assert"
)

func TestResolvingDefaultRegion(t *testing.T) {
	os.Setenv("AWS_REGION", "")
	os.Setenv("AWS_DEFAULT_REGION", "")
	region := amazon.ResolveRegion()
	assert.Equal(t, amazon.DefaultRegion, region)
}

func TestResolvingRegionFromAwsRegionEnv(t *testing.T) {
	os.Setenv("AWS_REGION", "us-east-1")
	region := amazon.ResolveRegion()
	assert.Equal(t, "us-east-1", region)
}

func TestResolvingRegionFromAwsDefaultRegionEnv(t *testing.T) {
	os.Setenv("DEFAULT_AWS_REGION", "us-east-1")
	region := amazon.ResolveRegion()
	assert.Equal(t, "us-east-1", region)
}

func TestReturnOption(t *testing.T) {
	region := amazon.ResolveRegionIfOptionEmpty("someRegion")
	assert.Equal(t, "someRegion", region)
}

func TestResolveIfOptionEmpty(t *testing.T) {
	os.Setenv("AWS_REGION", "us-east-1")
	region := amazon.ResolveRegionIfOptionEmpty("")
	assert.Equal(t, "us-east-1", region)
}