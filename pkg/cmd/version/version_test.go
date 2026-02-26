// +build unit

package version_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/v2/pkg/cmd/version"
	"github.com/stretchr/testify/assert"
)

var versiontests = []struct {
	short bool
	desc  string
}{
	{true, "Version with short flag"},
	{false, "Normal version"},
}

// Basic unit tests to check that all the different flags properly for jx version sub command
func Test_ExecuteVersion(t *testing.T) {
	for _, v := range versiontests {
		t.Run(v.desc, func(t *testing.T) {
			// fakeout the output for the tests
			out := &testhelpers.FakeOut{}
			commonOpts := opts.NewCommonOptionsWithTerm(clients.NewFactory(), os.Stdin, out, os.Stderr)

			// Set batchmode to true for tests
			commonOpts.BatchMode = true
			command := version.NewCmdVersion(commonOpts)

			switch v.short {
			case true:
				command.SetArgs([]string{"--short"})
				err := command.Execute()
				assert.NoError(t, err, "could not execute version")
				assert.Contains(t, out.GetOutput(), "Version")
				assert.NotContains(t, out.GetOutput(), "Commit")
				assert.NotContains(t, out.GetOutput(), "Build date")
				assert.NotContains(t, out.GetOutput(), "Go version")
				assert.NotContains(t, out.GetOutput(), "Git tree state")
			default:
				err := command.Execute()
				assert.NoError(t, err, "could not execute version")
				assert.Contains(t, out.GetOutput(), "Version")
				assert.Contains(t, out.GetOutput(), "Commit")
				assert.Contains(t, out.GetOutput(), "Build date")
				assert.Contains(t, out.GetOutput(), "Go version")
				assert.Contains(t, out.GetOutput(), "Git tree state")
			}
		})
	}
}
