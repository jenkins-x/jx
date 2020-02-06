// +build integration

package opts_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/log"
)

func TestCommonOptions_EnsureGCloudBinaryInstalled(t *testing.T) {
	o := opts.NewCommonOptionsWithFactory(clients.NewFactory())
	o.NoBrew = true

	origJXHome, testJXHome, err := testhelpers.CreateTestJxHomeDir()
	if err != nil {
		t.Errorf("Failed to create a test JX Home directory")
	}

	defer func() {
		err := testhelpers.CleanupTestJxHomeDir(origJXHome, testJXHome)
		if err != nil {
			log.Logger().Warnf("Unable to remove temporary directory %s: %s", testJXHome, err)
		}
	}()

	err = o.InstallGcloud()
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(testJXHome, "bin", "gcloud"))
}
