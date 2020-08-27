package plugins

import (
	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
)

const (
	// AdminVersion the version of the jx admin plugin
	AdminVersion = "0.0.83"

	// ApplicationVersion the version of the jx application plugin
	ApplicationVersion = "0.0.10"

	// GitOpsVersion the version of the jx gitops plugin
	GitOpsVersion = "0.0.240"

	// JenkinsVersion the version of the jx jenkins plugin
	JenkinsVersion = "0.0.22"

	// PipelineVersion the version of the jx pipeline plugin
	PipelineVersion = "0.0.14"

	// PreviewVersion the version of the jx preview plugin
	PreviewVersion = "0.0.24"

	// ProjectVersion the version of the jx project plugin
	ProjectVersion = "0.0.69"

	// PromoteVersion the version of the jx promote plugin
	PromoteVersion = "0.0.80"

	// SecretVersion the version of the jx secret plugin
	SecretVersion = "0.0.98"

	// TestVersion the version of the jx test plugin
	TestVersion = "0.0.18"

	// VerifyVersion the version of the jx verify plugin
	VerifyVersion = "0.0.18"
)

var (
	// Plugins default plugins
	Plugins = []jenkinsv1.Plugin{
		CreateJXPlugin("admin", AdminVersion),
		CreateJXPlugin("application", ApplicationVersion),
		CreateJXPlugin("gitops", GitOpsVersion),
		CreateJXPlugin("jenkins", JenkinsVersion),
		CreateJXPlugin("pipeline", PipelineVersion),
		CreateJXPlugin("preview", PreviewVersion),
		CreateJXPlugin("project", ProjectVersion),
		CreateJXPlugin("promote", PromoteVersion),
		CreateJXPlugin("secret", SecretVersion),
		CreateJXPlugin("test", TestVersion),
		CreateJXPlugin("verify", VerifyVersion),
	}
)
