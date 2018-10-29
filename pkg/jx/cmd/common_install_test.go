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
	assert.Nil(t, err)
	defer os.Setenv("PATH", oldPath)

	defer os.Unsetenv("JX_HOME")
	tempDir, err := ioutil.TempDir("", "common_install_test")
	err = os.Setenv("JX_HOME", tempDir)
	assert.Nil(t, err)
	err = (&CommonOptions{}).installEksCtl(false)
	eksctl := filepath.Join(os.Getenv("JX_HOME"), "/bin/eksctl")
	if runtime.GOOS == "windows" {
		eksctl += ".exe"
	}
	assert.FileExists(t, eksctl)
}
