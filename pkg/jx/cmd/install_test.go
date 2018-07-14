package cmd

import (
	"os"
	"path"
	"testing"

	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestInstall(t *testing.T) {
	testDir := path.Join("test_data", "install_cloud_environments_repo")
	_, err := os.Stat(testDir)
	assert.NoError(t, err)

	version, err := loadVersionFromCloudEnvironmentsDir(testDir)
	assert.NoError(t, err)

	assert.Equal(t, "0.0.1436", version, "For Makefile in dir %s", testDir)
}

func TestGenerateProwSecret(t *testing.T) {

	fmt.Println(util.RandStringBytesMaskImprSrc(41))
}
