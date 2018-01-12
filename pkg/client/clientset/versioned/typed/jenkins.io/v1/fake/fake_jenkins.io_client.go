package fake

import (
	v1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeJenkinsV1 struct {
	*testing.Fake
}

func (c *FakeJenkinsV1) Environments(namespace string) v1.EnvironmentInterface {
	return &FakeEnvironments{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeJenkinsV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
