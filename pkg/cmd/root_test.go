package cmd_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	rootCmd := cmd.Main([]string{""})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}
