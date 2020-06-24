// +build integration

package ksync_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/v2/pkg/ksync"
	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx-logging/pkg/log"
)

// TestInstallKsync tests that Ksync gets properly installed into JX_HOME
func TestInstallKsync(t *testing.T) {
	origJXHome, testJXHome, err := testhelpers.CreateTestJxHomeDir()
	assert.NoError(t, err, "failed to create a test JX Home directory")

	defer func() {
		err = testhelpers.CleanupTestJxHomeDir(origJXHome, testJXHome)
		if err != nil {
			log.Logger().Warnf("unable to remove temporary directory %s: %s", testJXHome, err)
		}
		log.Logger().Info("Deleted temporary jx home directory")
	}()

	// Test installing ksync locally
	_, err = ksync.InstallKSync()
	assert.NoError(t, err, "error installing ksync")
	assert.FileExists(t, filepath.Join(testJXHome, "bin", "ksync"))
}
