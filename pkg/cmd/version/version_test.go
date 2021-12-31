package version_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/version"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdVersion(t *testing.T) {
	cmd, _ := version.NewCmdVersion()
	err := cmd.Execute()
	assert.NoError(t, err)
}
