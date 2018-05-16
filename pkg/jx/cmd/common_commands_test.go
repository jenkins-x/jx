package cmd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExecuteCommand(t *testing.T) {
	o := CommonOptions{}
	err := o.runCommand("echo", "foo")
	assert.Nil(t, err)

}

func TestCommandError(t *testing.T) {
	o := CommonOptions{}
	err := o.runCommand("noSuchCommand")
	assert.NotNil(t, err)

}

func TestVerboseOutput(t *testing.T) {
	out := new(bytes.Buffer)
	o := CommonOptions{Verbose: true, Out: out}
	o.runCommand("echo", "foo")
	assert.Equal(t, out.String(), "foo\n")
}

func TestNonVerboseOutput(t *testing.T) {
	out := new(bytes.Buffer)
	o := CommonOptions{Out: out}
	o.runCommand("echo", "foo")
	assert.Empty(t, out.String())
}
