package amazon_test

import (
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func switchHome() (string, error) {
	oldHome := os.Getenv("HOME")
	newHome := "/tmp/" + uuid.New()
	os.Setenv("HOME", newHome)
	awsHome := path.Join(newHome, ".aws")
	err := os.MkdirAll(awsHome, 0777)
	if err != nil {
		return oldHome, err
	}

	awsConfigPath := path.Join(awsHome, "config")
	ioutil.WriteFile(awsConfigPath, []byte(`[profile foo]
region = bar
[profile baz]
region = qux
`), 0644)

	return oldHome, nil
}

func restoreHome(oldHome string) {
	os.Setenv("HOME", oldHome)
}

func configureEnv(region string, defaultRegion string, profile string) {
	os.Setenv("AWS_REGION", region)
	os.Setenv("AWS_DEFAULT_REGION", defaultRegion)
	os.Setenv("AWS_PROFILE", profile)
}

// Region tests

func TestResolvingDefaultRegion(t *testing.T) {
	t.Parallel()
	oldHome, err := switchHome()
	defer restoreHome(oldHome)
	assert.Nil(t, err)
	configureEnv("", "", "")
	session, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, amazon.DefaultRegion, *session.Config.Region)
}

func TestResolvingRegionFromAwsRegionEnv(t *testing.T) {
	t.Parallel()
	oldHome, err := switchHome()
	defer restoreHome(oldHome)
	assert.Nil(t, err)
	configureEnv("us-east-1", "", "")
	session, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "us-east-1", *session.Config.Region)
}

func TestResolvingRegionFromAwsDefaultRegionEnv(t *testing.T) {
	t.Parallel()
	oldHome, err := switchHome()
	defer restoreHome(oldHome)
	assert.Nil(t, err)
	configureEnv("", "us-east-1", "")
	session, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "us-east-1", *session.Config.Region)
}

func TestReturnOption(t *testing.T) {
	t.Parallel()
	oldHome, err := switchHome()
	defer restoreHome(oldHome)
	assert.Nil(t, err)
	configureEnv("", "", "")
	session, err := amazon.NewAwsSession("", "someRegion")
	assert.Nil(t, err)
	assert.Equal(t, "someRegion", *session.Config.Region)
}

func TestReadingRegionFromConfigProfile(t *testing.T) {
	t.Parallel()
	oldHome, err := switchHome()
	defer restoreHome(oldHome)
	assert.Nil(t, err)
	configureEnv("", "", "")
	session, err := amazon.NewAwsSession("foo", "")
	assert.Nil(t, err)
	assert.Equal(t, "bar", *session.Config.Region)
}

func TestReadingRegionFromEnvProfile(t *testing.T) {
	t.Parallel()
	oldHome, err := switchHome()
	defer restoreHome(oldHome)
	assert.Nil(t, err)
	configureEnv("", "", "baz")
	session, err := amazon.NewAwsSessionWithoutOptions()
	assert.Nil(t, err)
	assert.Equal(t, "qux", *session.Config.Region)
}
