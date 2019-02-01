package version_test

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/jenkins-x/jx/pkg/version"
	"github.com/stretchr/testify/assert"
)

// TODO refactor to encapsulate
func TestLoadVersionData(t *testing.T) {
	dataDir := "test_data/jenkins-x-versions"

	AssertLoadTestData(t, dataDir, version.KindPackage, "helm", "2.12.2")
	AssertLoadTestData(t, dataDir, version.KindChart, "jenkins-x/knative-build", "0.1.13")
	AssertLoadTestData(t, dataDir, version.KindChart, "doesNotExist", "")
}

// AssertLoadTestData asserts that the VersionData can be loaded/created for the given kind
func AssertLoadTestData(t *testing.T, dataDir string, kind version.VersionKind, name string, expectedValue string) {
	data, err := version.LoadVersionData(dataDir, kind, name)
	require.NoError(t, err, "failed to load VersionData for dir %s kind %s name %s", dataDir, string(kind), name)

	assert.Equal(t, expectedValue, data.Version, "wrong version for kind %s name %s", string(kind), name)
}
