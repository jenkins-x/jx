package kube

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToValidName(t *testing.T) {
	assertToValidName(t, "foo", "foo")
	assertToValidName(t, "foo-bar", "foo-bar")
	assertToValidName(t, "foo-bar-", "foo-bar")
	assertToValidName(t, "foo-bar-0.1.0", "foo-bar-0-1-0")
	assertToValidName(t, "---foo-bar-", "foo-bar")
	assertToValidName(t, "foo/bar_*123", "foo-bar-123")
}

func TestToValidNameWithDots(t *testing.T) {
	assertToValidNameWithDots(t, "foo-bar-0.1.0", "foo-bar-0.1.0")
	assertToValidNameWithDots(t, "foo/bar_.123", "foo-bar-.123")
}

func assertToValidNameWithDots(t *testing.T, input string, expected string) {
	actual := ToValidNameWithDots(input)
	assert.Equal(t, expected, actual, "ToValidNameWithDots for input %s", input)
}

func assertToValidName(t *testing.T, input string, expected string) {
	actual := ToValidName(input)
	assert.Equal(t, expected, actual, "ToValidName for input %s", input)
}
