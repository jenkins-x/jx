package amazon_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/stretchr/testify/assert"
)

func TestCreateNewSessionWithDefaultRegion(t *testing.T) {
	// TODO Refactor for encapsulation
	oldHome, err := switchHome()
	defer restoreHome(oldHome)
	os.Setenv("AWS_REGION", "")
	os.Setenv("AWS_DEFAULT_REGION", "")
	os.Setenv("AWS_PROFILE", "")
	sess, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "us-west-2", *sess.Config.Region)
}

func TestCreateNewSessionWithRegionFromAwsRegion(t *testing.T) {
	// TODO Refactor for encapsulation
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_DEFAULT_REGION", "")
	os.Setenv("AWS_PROFILE", "")
	sess, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "us-east-1", *sess.Config.Region)
}
