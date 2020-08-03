package plugins

import (
	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
)

const (
	// AdminVersion the version of the jx admin plugin
	AdminVersion = "0.0.57"

	// ApplicationVersion the version of the jx application plugin
	ApplicationVersion = "0.0.10"

	// GitOpsVersion the version of the jx gitops plugin
	GitOpsVersion = "0.0.162"

	// PipelineVersion the version of the jx pipeline plugin
	PipelineVersion = "0.0.14"

	// PreviewVersion the version of the jx preview plugin
	PreviewVersion = "0.0.24"

	// ProjectVersion the version of the jx project plugin
	ProjectVersion = "0.0.53"

	// PromoteVersion the version of the jx promote plugin
	PromoteVersion = "0.0.64"

	// SecretVersion the version of the jx secret plugin
	SecretVersion = "0.0.44"

	// VerifyVersion the version of the jx verify plugin
	VerifyVersion = "0.0.12"
)

var (
	// Plugins default plugins
	Plugins = []jenkinsv1.Plugin{
		CreateJXPlugin("admin", AdminVersion),
		CreateJXPlugin("application", ApplicationVersion),
		CreateJXPlugin("gitops", GitOpsVersion),
		CreateJXPlugin("pipeline", PipelineVersion),
		CreateJXPlugin("preview", PreviewVersion),
		CreateJXPlugin("project", ProjectVersion),
		CreateJXPlugin("promote", PromoteVersion),
		CreateJXPlugin("secret", SecretVersion),
		CreateJXPlugin("verify", VerifyVersion),
	}
)
