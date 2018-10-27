package amazon_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/stretchr/testify/assert"
)

func TestCreateNewSessionWithDefaultRegion(t *testing.T) {
	t.Parallel()
	// TODO Refactor for encapsulation
	os.Setenv("AWS_REGION", "")
	os.Setenv("AWS_DEFAULT_REGION", "")
	sess, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "us-west-2", *sess.Config.Region)
}

func TestCreateNewSessionWithRegionFromAwsRegion(t *testing.T) {
	// TODO Parallel should be called, but test fails when we do.
	// t.Parallel()
	// TODO Refactor for encapsulation
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_DEFAULT_REGION", "")
	sess, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "us-east-1", *sess.Config.Region)
}
