package amazon_test

import (
	"github.com/jenkins-x/jx/pkg/tests"
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/stretchr/testify/assert"
)

func TestCreateNewSessionWithDefaultRegion(t *testing.T) {
	tests.SkipForWindows(t, "Pre-existing test. Reason not investigated")
	// TODO Refactor for encapsulation
	os.Setenv("AWS_REGION", "")
	os.Setenv("AWS_DEFAULT_REGION", "")
	sess, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "us-west-2", *sess.Config.Region)
}

func TestCreateNewSessionWithRegionFromAwsRegion(t *testing.T) {
	tests.SkipForWindows(t, "Pre-existing test. Reason not investigated")
	// TODO Refactor for encapsulation
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_DEFAULT_REGION", "")
	sess, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "us-east-1", *sess.Config.Region)
}
