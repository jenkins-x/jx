package cmd_test

import (
	"bytes"
	"testing"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
)

func TestExecuteCommand(t *testing.T) {
	t.Parallel()
	o := cmd.CommonOptions{}
	err := o.RunCommand("echo", "foo")
	assert.Nil(t, err)
}

func TestCommandError(t *testing.T) {
	t.Parallel()
	o := cmd.CommonOptions{}
	err := o.RunCommand("noSuchCommand")
	assert.NotNil(t, err)
}

func TestVerboseOutput(t *testing.T) {
	t.Parallel()
	out := new(bytes.Buffer)
	o := cmd.CommonOptions{Verbose: true, Out: out}
	o.RunCommand("echo", "foo")
	assert.Equal(t, out.String(), "foo\n")
}

func TestNonVerboseOutput(t *testing.T) {
	t.Parallel()
	out := new(bytes.Buffer)
	o := cmd.CommonOptions{Out: out}
	o.RunCommand("echo", "foo")
	assert.Empty(t, out.String())
}
