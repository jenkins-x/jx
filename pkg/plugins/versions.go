package plugins

import (
	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
)

const (
	// AdminVersion the version of the jx admin plugin
	AdminVersion = "0.0.143"

	// ApplicationVersion the version of the jx application plugin
	ApplicationVersion = "0.0.19"

	// GitOpsVersion the version of the jx gitops plugin
	GitOpsVersion = "0.0.446"

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
	SecretVersion = "0.0.181"

	// TestVersion the version of the jx test plugin
	TestVersion = "0.0.21"

	// VerifyVersion the version of the jx verify plugin
	VerifyVersion = "0.0.34"
)

var (
	// Plugins default plugins
	Plugins = []jenkinsv1.Plugin{
		CreateJXPlugin(jenkinsxOrganisation, "admin", AdminVersion),
		CreateJXPlugin(jenkinsxOrganisation, "application", ApplicationVersion),
		CreateJXPlugin(jenkinsxOrganisation, "gitops", GitOpsVersion),
		CreateJXPlugin(jenkinsxPluginsOrganisation, "health", HealthVersion),
		CreateJXPlugin(jenkinsxOrganisation, "jenkins", JenkinsVersion),
		CreateJXPlugin(jenkinsxOrganisation, "pipeline", PipelineVersion),
		CreateJXPlugin(jenkinsxOrganisation, "preview", PreviewVersion),
		CreateJXPlugin(jenkinsxOrganisation, "project", ProjectVersion),
		CreateJXPlugin(jenkinsxOrganisation, "promote", PromoteVersion),
		CreateJXPlugin(jenkinsxOrganisation, "secret", SecretVersion),
		CreateJXPlugin(jenkinsxOrganisation, "test", TestVersion),
		CreateJXPlugin(jenkinsxOrganisation, "verify", VerifyVersion),
	}
)
