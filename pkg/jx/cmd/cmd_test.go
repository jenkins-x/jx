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
	cmd := cmd.NewJXCommand(f, os.Stdin, os.Stdout, os.Stderr, []string{"prompt"})
	// This hack allows us to override the command that will actually be executed.
	// Normally it would come directly from os.Args
	cmd.SetArgs([]string{"prompt"})
	err := cmd.Execute()
	assert.Nil(t, err)
}
