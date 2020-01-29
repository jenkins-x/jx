package envctx_test

import (
	"path"
	"testing"

	"github.com/jenkins-x/jx/pkg/envctx"
	"github.com/jenkins-x/jx/pkg/versionstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentContextResolveChartDetails(t *testing.T) {
	t.Parallel()

	versionsDir := path.Join("test_data", "jenkins-x-versions")
	assert.DirExists(t, versionsDir)

	ec := &envctx.EnvironmentContext{
		VersionResolver: &versionstream.VersionResolver{
			VersionsDir: versionsDir,
		},
	}

	type testData struct {
		Test       string
		Name       string
		Repository string
		Expected   envctx.ChartDetails
	}
	tests := []testData{
		{
			Test:       "findRepositoryFromPrefix",
			Name:       "jenkins-x/lighthouse",
			Repository: "",
			Expected: envctx.ChartDetails{
				Name:       "jenkins-x/lighthouse",
				Prefix:     "jenkins-x",
				LocalName:  "lighthouse",
				Repository: "https://storage.googleapis.com/chartmuseum.jenkins-x.io",
			},
		},
		{
			Test:       "findPrefixFromRepositoryURL",
			Name:       "lighthouse",
			Repository: "https://storage.googleapis.com/chartmuseum.jenkins-x.io",
			Expected: envctx.ChartDetails{
				Name:       "jenkins-x/lighthouse",
				Prefix:     "jenkins-x",
				LocalName:  "lighthouse",
				Repository: "https://storage.googleapis.com/chartmuseum.jenkins-x.io",
			},
		},
		{
			Test: "findPrefixFromAliasRepositoryURL",
			Name: "lighthouse",
			// lets try use an alias
			Repository: "http://chartmuseum.jenkins-x.io",
			Expected: envctx.ChartDetails{
				Name:       "jenkins-x/lighthouse",
				Prefix:     "jenkins-x",
				LocalName:  "lighthouse",
				Repository: "http://chartmuseum.jenkins-x.io",
			},
		},
		{
			Test:       "localChart",
			Name:       "repositories",
			Repository: "..",
			Expected: envctx.ChartDetails{
				Name:       "../repositories",
				Prefix:     "..",
				LocalName:  "repositories",
				Repository: "",
			},
		},
	}

	for _, test := range tests {
		name := test.Name
		repo := test.Repository
		expected := test.Expected
		actual, err := ec.ChartDetails(name, repo)
		require.NoError(t, err, "failed to find chart details for %s and %s", name, repo)

		assert.Equal(t, expected.Name, actual.Name, "chartDetails.Name for test %s", test.Test)
		assert.Equal(t, expected.LocalName, actual.LocalName, "chartDetails.LocalName for test %s", test.Test)
		assert.Equal(t, expected.Prefix, actual.Prefix, "chartDetails.Prefix for test %s", test.Test)
		assert.Equal(t, expected.Repository, actual.Repository, "chartDetails.Repository for test %s", test.Test)

	}
}
