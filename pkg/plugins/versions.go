package plugins

import (
	jenkinsv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
)

const (
	// GitOpsVersion the default version of the jx gitops plugin
	GitOpsVersion = "0.0.54"
)

var (
	// Plugins default plugins
	Plugins = []jenkinsv1.Plugin{
		CreateJXPlugin("gitops", GitOpsVersion),
	}
)
