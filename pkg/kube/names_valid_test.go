package kube

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToValidName(t *testing.T) {
	assertToValidName(t, "foo", "foo")
	assertToValidName(t, "foo-bar", "foo-bar")
	assertToValidName(t, "foo-bar-", "foo-bar")
	assertToValidName(t, "---foo-bar-", "foo-bar")
	assertToValidName(t, "foo/bar_*123", "foo-bar-123")
}

func assertToValidName(t *testing.T, input string, expected string) {
	actual := ToValidName(input)
	assert.Equal(t, expected, actual, "ToValidName for input %s", input)
}
