// +build unit

package version_test

import (
	"testing"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/v2/pkg/version"
	"github.com/stretchr/testify/assert"
)

var InputNonEmpty = map[string]string{
	"version":      "2.2.2",
	"revision":     "abcdef",
	"buildDate":    "20200526-19:28:09",
	"goVersion":    "1.14.8",
	"gitTreeState": "dirty",
}

var OutputNonEmpty = InputNonEmpty

var InputEmpty = map[string]string{
	"version":      "",
	"revision":     "",
	"buildDate":    "",
	"goVersion":    "",
	"gitTreeState": "",
}

var OutputEmpty = map[string]string{
	"version":      version.TestVersion,
	"revision":     version.TestRevision,
	"buildDate":    version.TestBuildDate,
	"goVersion":    version.TestGoVersion,
	"gitTreeState": version.TestTreeState,
}

var getterTests = []struct {
	description string
	in          map[string]string
	out         map[string]string
}{
	{
		"Case 1: build passes a value",
		InputNonEmpty,
		OutputNonEmpty,
	},
	{
		"Case 2: No value passed during build",
		InputEmpty,
		OutputEmpty,
	},
}

func TestGetters(t *testing.T) {
	for _, tt := range getterTests {
		t.Run(tt.description, func(t *testing.T) {
			version.Map["version"] = tt.in["version"]
			version.Map["revision"] = tt.in["revision"]
			version.Map["buildDate"] = tt.in["buildDate"]
			version.Map["goVersion"] = tt.in["goVersion"]
			version.Map["gitTreeState"] = tt.in["gitTreeState"]

			result := version.GetVersion()
			assert.Equal(t, tt.out["version"], result)

			result = version.GetRevision()
			assert.Equal(t, tt.out["revision"], result)

			result = version.GetBuildDate()
			assert.Equal(t, tt.out["buildDate"], result)

			result = version.GetGoVersion()
			assert.Equal(t, tt.out["goVersion"], result)

			result = version.GetTreeState()
			assert.Equal(t, tt.out["gitTreeState"], result)
		})
	}
}

// TODO refactor to encapsulate
func TestGetSemverVersisonWithStandardVersion(t *testing.T) {
	version.Map["version"] = "1.2.1"
	result, err := version.GetSemverVersion()
	expectedResult := semver.Version{Major: 1, Minor: 2, Patch: 1}
	assert.NoError(t, err, "GetSemverVersion should exit without failure")
	assert.Exactly(t, expectedResult, result)
}

// TODO refactor to encapsulate
func TestGetSemverVersisonWithNonStandardVersion(t *testing.T) {
	version.Map["version"] = "1.3.153-dev+7a8285f4"
	result, err := version.GetSemverVersion()

	prVersions := []semver.PRVersion{{VersionStr: "dev"}}
	builds := []string{"7a8285f4"}
	expectedResult := semver.Version{Major: 1, Minor: 3, Patch: 153, Pre: prVersions, Build: builds}
	assert.NoError(t, err, "GetSemverVersion should exit without failure")
	assert.Exactly(t, expectedResult, result)
}
