package cmd

import (
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestInstallEksctl(t *testing.T) {
	oldPath := os.Getenv("PATH")
	err := os.Setenv("PATH", "")
	assert.Nil(t, err)
	defer os.Setenv("PATH", oldPath)

	defer os.Unsetenv("JX_HOME")
	err = os.Setenv("JX_HOME", "/tmp/" + uuid.New())
	assert.Nil(t, err)
	err = (&CommonOptions{}).installEksCtl(false)
	assert.FileExists(t, os.Getenv("JX_HOME") + "/bin/eksctl")
}