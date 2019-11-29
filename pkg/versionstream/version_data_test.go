// +build unit

package versionstream_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/versionstream"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	dataDir = "test_data/jenkins-x-versions"
)

// TODO refactor to encapsulate
func TestLoadVersionData(t *testing.T) {
	AssertLoadTestData(t, dataDir, versionstream.KindChart, "jenkins-x/knative-build", "0.1.13")
	AssertLoadTestData(t, dataDir, versionstream.KindChart, "doesNotExist", "")
	AssertLoadTestData(t, dataDir, versionstream.KindPackage, "helm", "2.12.2")
}

// AssertLoadTestData asserts that the StableVersion can be loaded/created for the given kind
func AssertLoadTestData(t *testing.T, dataDir string, kind versionstream.VersionKind, name string, expectedValue string) {
	data, err := versionstream.LoadStableVersion(dataDir, kind, name)
	require.NoError(t, err, "failed to load StableVersion for dir %s kind %s name %s", dataDir, string(kind), name)

	assert.Equal(t, expectedValue, data.Version, "wrong version for kind %s name %s", string(kind), name)
}

// TestExactPackage tests an exact package version
func TestExactPackage(t *testing.T) {
	resolver := &versionstream.VersionResolver{
		VersionsDir: dataDir,
	}

	AssertPackageVersion(t, resolver, "helm", "2.12.2", true)
	AssertPackageVersion(t, resolver, "helm", "2.12.3", false)
}

// TestRepositories tests we can load the repository prefix -> URL maps
func TestRepositories(t *testing.T) {

	prefixes, err := versionstream.GetRepositoryPrefixes(dataDir)
	require.NoError(t, err, "GetRepositoryPrefixes() failed on dir %s", dataDir)

	data := map[string]string{
		"https://storage.googleapis.com/chartmuseum.jenkins-x.io": "jenkins-x",
		"http://chartmuseum.jenkins-x.io":                         "jenkins-x",
		"https://kubernetes-charts.storage.googleapis.com":        "stable",
	}
	for u, p := range data {
		assert.Equal(t, p, prefixes.PrefixForURL(u), "failed to find correct repository prefix for URL %s", u)
	}
}

// TestExactPackageVersionRange tests ranges of packages
func TestExactPackageVersionRange(t *testing.T) {
	resolver := &versionstream.VersionResolver{
		VersionsDir: dataDir,
	}

	AssertPackageVersion(t, resolver, "kubectl", "1.12.0", true)
	AssertPackageVersion(t, resolver, "kubectl", "1.12.1", true)
	AssertPackageVersion(t, resolver, "kubectl", "1.13.1", true)

	AssertPackageVersion(t, resolver, "kubectl", "v1.13.1", true)

	AssertPackageVersion(t, resolver, "kubectl", "1.10.1", false)
	AssertPackageVersion(t, resolver, "kubectl", "2.0.0", false)
	AssertPackageVersion(t, resolver, "kubectl", "2.0.1", false)

	AssertPackageVersion(t, resolver, "git", "2.1.1 (Apple Git-117)", false)
	AssertPackageVersion(t, resolver, "git", "2.20.1 (Apple Git-117)", true)
}

func AssertPackageVersion(t *testing.T, resolver *versionstream.VersionResolver, name string, version string, expectedValid bool) {
	err := resolver.VerifyPackage(name, version)
	if expectedValid {
		assert.NoError(t, err, "expected a valid version %s for package %s", version, name)
	} else {
		t.Logf("got expected error %s\n", err.Error())
		assert.Error(t, err, "expected an invalid version %s for package %s", version, name)
	}
}

func TestResolveDockerImage(t *testing.T) {
	var testCases = []struct {
		dataDir               string
		resolveImage          string
		expectedResolvedImage string
		expectError           bool
		errorMessage          string
	}{
		{"foo", "foo", "foo", false, ""},
		{dataDir, "foo", "foo", false, ""},
		{dataDir, "builder-jx", "builder-jx", false, ""},
		{dataDir, "jenkinsxio/builder-jx", "jenkinsxio/builder-jx", false, ""},
		{dataDir, "gcr.io/jenkinsxio/builder-jx", "gcr.io/jenkinsxio/builder-jx:1.0.0", false, ""},
		{dataDir, "docker.io/fubar", "fubar:2.0.0", false, ""},
		{dataDir, "docker.io/snafu", "snafu", false, ""},
		{dataDir, "susfu", "susfu", true, "failed to unmarshal YAML"},
	}

	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("test_resolve_%s", testCase.resolveImage), func(t *testing.T) {
			actualResolvedImage, err := versionstream.ResolveDockerImage(dataDir, testCase.resolveImage)
			if testCase.expectError {
				assert.Error(t, err, "expected call to ResolveDockerImage to fail")
				assert.Contains(t, err.Error(), testCase.errorMessage, "error message does not match")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedResolvedImage, actualResolvedImage, "image was not resolved as expected.")
			}
		})
	}
}

// TestGitURLToName tests version.GitURLToName()
func TestGitURLToName(t *testing.T) {
	data := map[string]string{
		"https://github.com/jenkins-x-buildpacks/jenkins-x-kubernetes":     "github.com/jenkins-x-buildpacks/jenkins-x-kubernetes",
		"https://github.com/jenkins-x-buildpacks/jenkins-x-kubernetes.git": "github.com/jenkins-x-buildpacks/jenkins-x-kubernetes",
		"http://github.com/jenkins-x-buildpacks/jenkins-x-kubernetes/":     "github.com/jenkins-x-buildpacks/jenkins-x-kubernetes",
	}
	for gitURL, expected := range data {
		actual := versionstream.GitURLToName(gitURL)
		assert.Equal(t, expected, actual, "GitURLToName for %s", gitURL)
	}
}
