package cmd_test

import (
	"os"
	"testing"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
)

func TestNewJXCommand(t *testing.T) {
	// TODO use mock stdio and expect some output ending in '>'
	f := cmd.NewFactory()
	f.SetOffline(true)
	cmd := cmd.NewJXCommand(f, os.Stdin, os.Stdout, os.Stderr)
	cmd.SetArgs([]string{"prompt"})
	err := cmd.Execute()
	assert.Nil(t, err)
}
