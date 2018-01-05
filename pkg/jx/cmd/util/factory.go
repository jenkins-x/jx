package util

import "github.com/jenkins-x/jx/pkg/jenkins"
import "github.com/jenkins-x/golang-jenkins"

type Factory interface {
	 GetJenkinsClient() (*gojenkins.Jenkins, error)
}


type factory struct {
}

// NewFactory creates a factory with the default Kubernetes resources defined
// if optionalClientConfig is nil, then flags will be bound to a new clientcmd.ClientConfig.
// if optionalClientConfig is not nil, then this factory will make use of it.
func NewFactory() Factory {
	return &factory{
	}
}

// GetJenkinsClient creates a new jenkins client
func (* factory) GetJenkinsClient() (*gojenkins.Jenkins, error) {
	return jenkins.GetJenkinsClient()
}
