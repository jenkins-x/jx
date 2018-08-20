package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteCommand(t *testing.T) {
	t.Parallel()
	o := CommonOptions{}
	err := o.runCommand("echo", "foo")
	assert.Nil(t, err)

}

func TestCommandError(t *testing.T) {
	t.Parallel()
	o := CommonOptions{}
	err := o.runCommand("noSuchCommand")
	assert.NotNil(t, err)

}

func TestVerboseOutput(t *testing.T) {
	t.Parallel()
	out := new(bytes.Buffer)
	o := CommonOptions{Verbose: true, Out: out}
	o.runCommand("echo", "foo")
	assert.Equal(t, out.String(), "foo\n")
}

func TestNonVerboseOutput(t *testing.T) {
	t.Parallel()
	out := new(bytes.Buffer)
	o := CommonOptions{Out: out}
	o.runCommand("echo", "foo")
	assert.Empty(t, out.String())
}
