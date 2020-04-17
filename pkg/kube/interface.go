package kube

import (
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Kuber defines common kube actions used within Jenkins X
//go:generate pegomock generate github.com/jenkins-x/jx/v2/pkg/kube Kuber -o mocks/kuber.go
type Kuber interface {
	// LoadConfig loads the Kubernetes configuration
	LoadConfig() (*api.Config, *clientcmd.PathOptions, error)

	// UpdateConfig defines new config entries for jx
	UpdateConfig(namespace string, server string, caData string, user string, token string) error
}
