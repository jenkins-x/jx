package opts_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/stretchr/testify/assert"
)

func TestInstallEksctl(t *testing.T) {
	oldPath := os.Getenv("PATH")
	err := os.Setenv("PATH", "")
	assert.NoError(t, err)
	defer os.Setenv("PATH", oldPath)

	originalJx, newJx, err := cmd.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := cmd.CleanupTestJxHomeDir(originalJx, newJx)
		assert.NoError(t, err)
	}()
	err = (&opts.CommonOptions{}).InstallEksCtl(false)
	assert.NoError(t, err)
	eksctl := filepath.Join(os.Getenv("JX_HOME"), "/bin/eksctl")
	if runtime.GOOS == "windows" {
		eksctl += ".exe"
	}
	assert.FileExists(t, eksctl)
}
