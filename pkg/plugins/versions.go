package plugins

import (
	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
)

const (
	// AdminVersion the version of the jx admin plugin
	AdminVersion = "0.0.190"

	// ApplicationVersion the version of the jx application plugin
	ApplicationVersion = "0.0.35"

	// GitOpsVersion the version of the jx gitops plugin
	GitOpsVersion = "0.3.3"

	// HealthVersion the version of the jx health plugin
	HealthVersion = "0.0.75"

	// JenkinsVersion the version of the jx jenkins plugin
	JenkinsVersion = "0.0.29"

	// OctantVersion the default version of octant to use
	OctantVersion = "0.21.0"

	// OctantJXVersion the default version of octant-jx plugin to use
	OctantJXVersion = "0.0.44"

	// PipelineVersion the version of the jx pipeline plugin
	PipelineVersion = "0.0.151"

	// PreviewVersion the version of the jx preview plugin
	PreviewVersion = "0.0.177"

	// ProjectVersion the version of the jx project plugin
	ProjectVersion = "0.2.11"

	// PromoteVersion the version of the jx promote plugin
	PromoteVersion = "0.0.274"

	// SecretVersion the version of the jx secret plugin
	SecretVersion = "0.1.48"

	// TestVersion the version of the jx test plugin
	TestVersion = "0.0.48"

	// VerifyVersion the version of the jx verify plugin
	VerifyVersion = "0.1.4"
)

var (
	// Plugins default plugins
	Plugins = []jenkinsv1.Plugin{
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "admin", AdminVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "application", ApplicationVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "gitops", GitOpsVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "health", HealthVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "pipeline", PipelineVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "preview", PreviewVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "project", ProjectVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "promote", PromoteVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "secret", SecretVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "test", TestVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "verify", VerifyVersion),
	}

	// PluginMap a map of plugin names like `jx-gitops` to the Plugin object
	PluginMap = map[string]*jenkinsv1.Plugin{}
)

func init() {
	for i := range Plugins {
		plugin := &Plugins[i]
		PluginMap[plugin.Spec.Name] = plugin
	}
}
