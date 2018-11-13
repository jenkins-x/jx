package cmd

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallEksctl(t *testing.T) {
	oldPath := os.Getenv("PATH")
	err := os.Setenv("PATH", "")
	assert.NoError(t, err)
	defer os.Setenv("PATH", oldPath)

	defer os.Unsetenv("JX_HOME")
	tempDir, err := ioutil.TempDir("", "common_install_test")
	err = os.Setenv("JX_HOME", tempDir)
	assert.NoError(t, err)
	err = (&CommonOptions{}).installEksCtl(false)
	assert.NoError(t, err)
	eksctl := filepath.Join(os.Getenv("JX_HOME"), "/bin/eksctl")
	if runtime.GOOS == "windows" {
		eksctl += ".exe"
	}
	assert.FileExists(t, eksctl)
}
