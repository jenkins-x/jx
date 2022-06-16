package upgrade_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/upgrade"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdUpgrade(t *testing.T) {
	cmd, _ := upgrade.NewCmdUpgrade()
	err := cmd.Execute()
	assert.NoError(t, err)
}
