package gits_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/stretchr/testify/assert"
)

type brancbNameData struct {
	input    string
	expected string
}

func Test(t *testing.T) {
	t.Parallel()
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
	git := &gits.GitCLI{}
	for _, data := range testCases {
		actual := git.ConvertToValidBranchName(data.input)
		assert.Equal(t, data.expected, actual, "Convert to valid branch name for %s", data.input)
	}
}
