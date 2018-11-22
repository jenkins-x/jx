package tests

import (
	"github.com/Netflix/go-expect"
	"github.com/acarl005/stripansi"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"testing"
)

// ConsoleWrapper is a wrapper around the go-expect Console that takes a test object and will report failures
// This prevents users having to manually detect and report errors during the tests
type ConsoleWrapper struct {
	tester  *testing.T
	console *expect.Console
	state   *vt10x.State
	terminal.Stdio
}

// ExpectString expects a string to be present on the console and fails the test if it is not
func (c *ConsoleWrapper) ExpectString(s string) {
	out, err := c.console.ExpectString(s)
	assert.NoError(c.tester, err, "Expected string: %q\nActual string: %q", s, stripansi.Strip(out))
}

// SendLine sends a string to the console and fails the test if something goes wrong
func (c *ConsoleWrapper) SendLine(s string) {
	_, err := c.console.SendLine(s)
	assert.NoError(c.tester, err, "Error sending line %s", s)
}

// ExpectEOF expects an EOF to be present on the console and reports an error if it is not
func (c *ConsoleWrapper) ExpectEOF() {
	out, err := c.console.ExpectEOF()
	assert.NoError(c.tester, err, "Expected EOF. Got %q", stripansi.Strip(out))
}

// Close closes the console
func (c *ConsoleWrapper) Close() error {
	return c.console.Tty().Close()
}

// CurrentState gets the last line of text currently on the console
func (c *ConsoleWrapper) CurrentState() string {
	return c.state.String()
}
