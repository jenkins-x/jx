package version_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/version"
	"github.com/stretchr/testify/assert"
)

// define a strtuct for test cases for the version command
type TestCase struct {
	args        []string
	err         error
	description string
	out         string
}

var testCases = []TestCase{
	{
		args:        nil,
		err:         nil,
		description: "This is to test without flags",
		out:         "version: 3.2.238\nshaCommit: 04b628f48\nbuildDate: 2022-05-31T14:51:38Z\ngoVersion: 1.17.8\nbranch: main\ngitTreeState: clean\n",
	},
	{
		args:        []string{"-q"},
		err:         nil,
		description: "This is to test -q flag",
		out:         "The --quit, -q flag is being deprecated from JX on Oct 2022\nUse --short, -s instead\n3.2.238\n",
	},
	{
		args:        []string{"--quiet"},
		err:         nil,
		description: "This is to test --quiet flag",
		out:         "The --quit, -q flag is being deprecated from JX on Oct 2022\nUse --short, -s instead\n3.2.238\n",
	},
	{
		args:        []string{"-s"},
		err:         nil,
		description: "This is to test -s flag",
		out:         "3.2.238\n",
	},
	{
		args:        []string{"--short"},
		err:         nil,
		description: "This is to test --short flag",
		out:         "3.2.238\n",
	},
}

func TestNewCmdVersion(t *testing.T) {
	for _, testCase := range testCases {
		var buf bytes.Buffer
		cmd, options := version.NewCmdVersion()
		options.Out = &buf
		cmd.SetArgs(testCase.args)
		err := cmd.Execute()

		fmt.Println(testCase.description)
		assert.NoError(t, err)
		if testCase.err == nil {
			assert.Equal(t, testCase.out, buf.String())
		}
	}
}
