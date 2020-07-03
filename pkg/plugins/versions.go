package plugins

import (
	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
)

const (
	// AdminVersion the version of the jx admin plugin
	AdminVersion = "0.0.35"

	// GitOpsVersion the version of the jx gitops plugin
	GitOpsVersion = "0.0.54"

	// ProjectVersion the version of the jx project plugin
	ProjectVersion = "0.0.17"

	// PromoteVersion the version of the jx promote plugin
	PromoteVersion = "0.0.54"

	// SecretVersion the version of the jx secret plugin
	SecretVersion = "0.0.36"
)

var (
	// Plugins default plugins
	Plugins = []jenkinsv1.Plugin{
		CreateJXPlugin("admin", AdminVersion),
		CreateJXPlugin("gitops", GitOpsVersion),
		CreateJXPlugin("project", ProjectVersion),
		CreateJXPlugin("promote", PromoteVersion),
		CreateJXPlugin("secret", SecretVersion),
	}
)
