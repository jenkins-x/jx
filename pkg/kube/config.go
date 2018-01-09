package kube

import (
	"k8s.io/client-go/tools/clientcmd/api"
)

// CurrentNamespace returns the current namespace in the context
func CurrentNamespace(config *api.Config) string {
	ctx := CurrentContext(config)
	if ctx != nil {
		return ctx.Namespace
	}
	return ""
}

// CurrentContext returns the current context
func CurrentContext(config *api.Config) *api.Context {
	if config != nil {
		name := config.CurrentContext
		if name != "" && config.Contexts != nil {
			return config.Contexts[name]
		}
	}
	return nil
}
