package naming_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/kube/naming"
	"github.com/stretchr/testify/assert"
)

func TestToValidName(t *testing.T) {
	t.Parallel()
	assertToValidName(t, "foo", "foo")
	assertToValidName(t, "foo-bar", "foo-bar")
	assertToValidName(t, "foo-bar-", "foo-bar")
	assertToValidName(t, "foo-bar-0.1.0", "foo-bar-0-1-0")
	assertToValidName(t, "---foo-bar-", "foo-bar")
	assertToValidName(t, "foo/bar_*123", "foo-bar-123")
}

func TestToValidNameWithDots(t *testing.T) {
	t.Parallel()
	assertToValidNameWithDots(t, "foo-bar-0.1.0", "foo-bar-0.1.0")
	assertToValidNameWithDots(t, "foo/bar_.123", "foo-bar-.123")
}

func TestToValidNameTruncated(t *testing.T) {
	t.Parallel()
	assertToValidNameTruncated(t, "foo", 7, "foo")
	assertToValidNameTruncated(t, "foo", 3, "foo")
	assertToValidNameTruncated(t, "foo", 2, "fo")
	assertToValidNameTruncated(t, "foo-bar", 4, "foo")
	assertToValidNameTruncated(t, "foo-bar", 5, "foo-b")
	assertToValidNameTruncated(t, "foo-bar-0.1.0", 10, "foo-bar-0")
	assertToValidNameTruncated(t, "---foo-bar-", 10, "foo-bar")
	assertToValidNameTruncated(t, "foo/bar_*123", 11, "foo-bar-123")
}

func assertToValidNameWithDots(t *testing.T, input string, expected string) {
	actual := naming.ToValidNameWithDots(input)
	assert.Equal(t, expected, actual, "ToValidNameWithDots for input %s", input)
}

func assertToValidName(t *testing.T, input string, expected string) {
	actual := naming.ToValidName(input)
	assert.Equal(t, expected, actual, "ToValidName for input %s", input)
}

func assertToValidNameTruncated(t *testing.T, input string, maxLength int, expected string) {
	actual := naming.ToValidNameTruncated(input, maxLength)
	assert.Equal(t, expected, actual, "ToValidNameTruncated for input %s", input)
}
