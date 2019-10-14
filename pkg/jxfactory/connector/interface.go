package connector

import (
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"k8s.io/client-go/rest"
)

// Client a client for connecting to remote clusters to help support
// multiple clusters for Environments
type Client interface {
	Connect(connector *v1.RemoteConnector) (*rest.Config, error)
}
