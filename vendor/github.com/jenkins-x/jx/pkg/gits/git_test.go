package gits

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type brancbNameData struct {
	input    string
	expected string
}

func Test(t *testing.T) {
	testCases := []brancbNameData{
		{
			"testing-thingy", "testing-thingy",
		},
		{
			"testing-thingy/", "testing-thingy",
		},
		{
			"testing-thingy.lock", "testing-thingy",
		},
		{
			"foo bar", "foo_bar",
		},
		{
			"foo\t ~bar", "foo_bar",
		},
	}
	for _, data := range testCases {
		actual := ConvertToValidBranchName(data.input)
		assert.Equal(t, data.expected, actual, "Convert to valid branch name for %s", data.input)
	}
}
