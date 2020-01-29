package envctx

import (
	"path/filepath"
	"strings"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/versionstream"
)

var (
	valuesFileNames = []string{"values.yaml", "values.yaml.gotmpl"}
)

// EnvironmentContext contains the common interfaces and structs needed for working with a development environment
type EnvironmentContext struct {
	// GitOps whether we are using gitops to manage this environment
	GitOps bool

	// Requirements the installation requirements
	Requirements *config.RequirementsConfig

	// DevEnv the development environment
	DevEnv *v1.Environment

	// VersionResolver the resolver of versions in the version stream
	VersionResolver *versionstream.VersionResolver
}

// TeamSettings returns the team settings for the current environment
func (c *EnvironmentContext) TeamSettings() *v1.TeamSettings {
	if c.DevEnv == nil {
		return nil
	}
	return &c.DevEnv.Spec.TeamSettings
}

// ChartDetails returns the chart details for the given chart name and repository URL
type ChartDetails struct {
	Name       string
	Prefix     string
	LocalName  string
	Repository string
}

// ChartDetails resolves the chart details from a full or local name and an optional repository URL.
// this function can handle an empty repository but the chart name "foo/bar" and resolve the prefix "foo" to a repository
// URL - or taking chart name "bar" and a repository URL and defaulting the prefix to "foo/bar"
func (c *EnvironmentContext) ChartDetails(chartName string, repo string) (*ChartDetails, error) {
	prefix := ""
	localName := chartName
	name := chartName
	paths := strings.SplitN(name, "/", 2)
	if len(paths) == 2 {
		localName = paths[1]
		prefix = paths[0]

		prefixes, err := c.VersionResolver.GetRepositoryPrefixes()
		if err != nil {
			return nil, err
		}
		urls := prefixes.URLsForPrefix(prefix)
		if len(urls) > 0 {
			repo = urls[0]
		}
	}
	teamSettings := c.TeamSettings()
	if repo == "" && teamSettings != nil {
		repo = teamSettings.AppsRepository
	}
	if repo == "" {
		repo = kube.DefaultChartMuseumURL
	}
	if prefix == "" {
		prefixes, err := c.VersionResolver.GetRepositoryPrefixes()
		if err != nil {
			return nil, err
		}
		prefix = prefixes.PrefixForURL(repo)
	}
	if prefix != "" && name == localName {
		name = prefix + "/" + name
	}

	// for local charts use the dir as the name
	if strings.HasPrefix(repo, ".") || strings.HasPrefix(repo, "/") {
		name = filepath.Join(repo, localName)
		repo = ""
		prefix = filepath.Dir(name)
	}
	return &ChartDetails{
		Name:       name,
		Prefix:     prefix,
		LocalName:  localName,
		Repository: repo,
	}, nil
}

// ResolveApplicationDefaults resolves the application defaults in the version stream if there are any
func (c *EnvironmentContext) ResolveApplicationDefaults(chartName string) (*config.ApplicationDefaultsConfig, []string, error) {
	valueFiles := []string{}
	dir := filepath.Join(c.VersionResolver.VersionsDir, string(versionstream.KindApp), chartName)
	defaults, _, err := config.LoadApplicationDefaultsConfig(dir)
	if err != nil {
		return defaults, valueFiles, err
	}

	// list the values files
	for _, f := range valuesFileNames {
		fileName := filepath.Join(dir, f)
		exists, _ := util.FileExists(fileName)
		if exists {
			valueFiles = append(valueFiles, fileName)
		}
	}
	return defaults, valueFiles, nil
}
