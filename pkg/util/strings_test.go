package util

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

type regexSplitData struct {
	input    string
	separator    string
	expected []string
}

func TestRegexpSplit(t *testing.T) {
	testCases := []regexSplitData{
		{
			"foo/bar", ":|/", []string{"foo", "bar"},
		},
		{
			"foo:bar", ":|/", []string{"foo", "bar"},
		},
	}
	for _, data := range testCases {
		actual := RegexpSplit(data.input, data.separator)
		assert.Equal(t, data.expected, actual, "Split did not match for input %s with separator %s", data.input, data.separator)
		//t.Logf("split %s with separator %s into %#v", data.input, data.separator, actual)
	}
}


func TestStringIndices(t *testing.T) {
	assertStringIndices(t, "foo/bar", "/", []int{3})
	assertStringIndices(t, "/foo/bar", "/", []int{0, 4})
}

func assertStringIndices(t *testing.T, text string, sep string, expected []int) {
	actual := StringIndexes(text, sep)
	assert.Equal(t, expected, actual, "Failed to evaluate StringIndices(%s, %s)", text, sep)
}
