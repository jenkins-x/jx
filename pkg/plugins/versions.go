package plugins

import (
	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
)

const (
	// AdminVersion the version of the jx admin plugin
	AdminVersion = "0.0.143"

	// ApplicationVersion the version of the jx application plugin
	ApplicationVersion = "0.0.19"

	// GitOpsVersion the version of the jx gitops plugin
	GitOpsVersion = "0.0.461"

	// HealthVersion the version of the jx health plugin
	HealthVersion = "0.0.57"

	// JenkinsVersion the version of the jx jenkins plugin
	JenkinsVersion = "0.0.29"

	// OctantVersion the default version of octant to use
	OctantVersion = "0.16.1"

	// OctantJXVersion the default version of octant-jx plugin to use
	OctantJXVersion = "0.0.34"

	// PipelineVersion the version of the jx pipeline plugin
	PipelineVersion = "0.0.71"

	// PreviewVersion the version of the jx preview plugin
	PreviewVersion = "0.0.128"

	// ProjectVersion the version of the jx project plugin
	ProjectVersion = "0.0.154"

	// PromoteVersion the version of the jx promote plugin
	PromoteVersion = "0.0.148"

	// SecretVersion the version of the jx secret plugin
	SecretVersion = "0.0.182"

	// TestVersion the version of the jx test plugin
	TestVersion = "0.0.21"

	// VerifyVersion the version of the jx verify plugin
	VerifyVersion = "0.0.34"
)

var (
	// Plugins default plugins
	Plugins = []jenkinsv1.Plugin{
		extensions.CreateJXPlugin(jenkinsxOrganisation, "admin", AdminVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "application", ApplicationVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "gitops", GitOpsVersion),
		extensions.CreateJXPlugin(jenkinsxPluginsOrganisation, "health", HealthVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "jenkins", JenkinsVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "pipeline", PipelineVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "preview", PreviewVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "project", ProjectVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "promote", PromoteVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "secret", SecretVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "test", TestVersion),
		extensions.CreateJXPlugin(jenkinsxOrganisation, "verify", VerifyVersion),
	}
)
