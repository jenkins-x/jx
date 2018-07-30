package kube

import (
	"fmt"
	"io/ioutil"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	DefaultNamespace = "jx"

	PodNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// LoadConfig loads the kubernetes configuration
func LoadConfig() (*api.Config, *clientcmd.PathOptions, error) {
	po := clientcmd.NewDefaultPathOptions()
	if po == nil {
		return nil, po, fmt.Errorf("Could not find any default path options for the kubeconfig file usually found at ~/.kube/config")
	}
	config, err := po.GetStartingConfig()
	if err != nil {
		return nil, po, fmt.Errorf("Could not load the kube config file %s due to %s", po.GetDefaultFilename(), err)
	}
	return config, po, err
}

// CurrentNamespace returns the current namespace in the context
func CurrentNamespace(config *api.Config) string {
	ctx := CurrentContext(config)
	if ctx != nil {
		n := ctx.Namespace
		if n != "" {
			return n
		}
	}
	// if we are in a pod lets try load the pod namespace file
	data, err := ioutil.ReadFile(PodNamespaceFile)
	if err == nil {
		n := string(data)
		if n != "" {
			return n
		}
	}
	return "default"
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

// CurrentServer returns the current context's server
func CurrentServer(config *api.Config) string {
	context := CurrentContext(config)
	return Server(config, context)
}

// Server returns the server of the given context
func Server(config *api.Config, context *api.Context) string {
	if context != nil && config != nil && config.Clusters != nil {
		cluster := config.Clusters[context.Cluster]
		if cluster != nil {
			return cluster.Server
		}
	}
	return ""
}
