// +build integration

package envctx_test

import (
	"path"
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/envctx"
	"github.com/jenkins-x/jx/pkg/versionstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentContextResolveChartDetails(t *testing.T) {
	t.Parallel()

	ec := createTestEnvironmentContext(t)

	appsConfig := &config.AppConfig{}

	type testData struct {
		Test       string
		Name       string
		Repository string
		Expected   envctx.ChartDetails
		AppsConfig *config.AppConfig
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
		{
			Test: "findPrefixFromAppsConfig",
			Name: "mydemo",
			// lets try use an alias
			Repository: "http://chartmuseum-jx.myinstall.com",
			AppsConfig: appsConfig,
			Expected: envctx.ChartDetails{
				Name:       "dev/mydemo",
				Prefix:     "dev",
				LocalName:  "mydemo",
				Repository: "http://chartmuseum-jx.myinstall.com",
			},
		},
		{
			Test: "findPrefixFromAppsConfigWithDifferentChartMusem",
			Name: "mydemo",
			// lets try use an alias
			Repository: "http://chartmuseum-anotherteam.myinstall.com",
			AppsConfig: appsConfig,
			Expected: envctx.ChartDetails{
				Name:       "dev2/mydemo",
				Prefix:     "dev2",
				LocalName:  "mydemo",
				Repository: "http://chartmuseum-anotherteam.myinstall.com",
			},
		},
	}

	for _, test := range tests {
		name := test.Name
		repo := test.Repository
		expected := test.Expected
		actual, err := ec.ChartDetails(name, repo)
		require.NoError(t, err, "failed to find chart details for %s and %s", name, repo)

		if test.AppsConfig != nil {
			actual.DefaultPrefix(test.AppsConfig, "dev")
		}
		assert.Equal(t, expected.Name, actual.Name, "chartDetails.Name for test %s", test.Test)
		assert.Equal(t, expected.LocalName, actual.LocalName, "chartDetails.LocalName for test %s", test.Test)
		assert.Equal(t, expected.Prefix, actual.Prefix, "chartDetails.Prefix for test %s", test.Test)
		assert.Equal(t, expected.Repository, actual.Repository, "chartDetails.Repository for test %s", test.Test)

		if test.AppsConfig != nil {
			found := false
			for _, r := range test.AppsConfig.Repositories {
				if r.URL == repo {
					t.Logf("the repository %s has been added to the appsConfig.repositories", repo)
					found = true
					break
				}
			}
			assert.Equal(t, true, found, "could not find repository %s in the appsConfig.repositories", repo)
		}
	}
}

func TestEnvironmentContextResolveApplicationDefaults(t *testing.T) {
	t.Parallel()

	ec := createTestEnvironmentContext(t)

	chartName := "stable/nginx-ingress"
	details, valuesFiles, err := ec.ResolveApplicationDefaults(chartName)
	require.NoError(t, err, "failed to resolve application defaults for chart %s", chartName)
	assert.Equal(t, len(valuesFiles), 1, "should have a values file")
	assert.Equal(t, "system", details.Phase, "details.Phase")
	assert.Equal(t, "nginx", details.Namespace, "details.Namespace")

	t.Logf("found details %#v and values files %#v\n", details, valuesFiles)
}

func createTestEnvironmentContext(t *testing.T) *envctx.EnvironmentContext {
	versionsDir := path.Join("test_data", "jenkins-x-versions")
	assert.DirExists(t, versionsDir)

	ec := &envctx.EnvironmentContext{
		VersionResolver: &versionstream.VersionResolver{
			VersionsDir: versionsDir,
		},
	}
	return ec
}
