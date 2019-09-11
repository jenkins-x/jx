package fake

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/cluster"
	"github.com/jenkins-x/jx/pkg/log"
)

// FakeClient a fake implementation of the cluster client
type FakeClient struct {
	Clusters []*cluster.Cluster
}

// verify we implement the interface
var _ cluster.Client = &FakeClient{}

// NewFake create a new fake client for testing
func NewClient(clusters []*cluster.Cluster) *FakeClient {
	return &FakeClient{
		Clusters: clusters,
	}
}

// List lists the clusters
func (c *FakeClient) List() ([]*cluster.Cluster, error) {
	return c.Clusters, nil
}

// ListFilter lists the clusters with a filter
func (c *FakeClient) ListFilter(labels map[string]string) ([]*cluster.Cluster, error) {
	return cluster.ListFilter(c, labels)
}

// Connect connects to a cluster
func (c *FakeClient) Connect(name string) error {
	log.Logger().Infof("fake cluster connecting to %s", name)
	return nil
}

// String return the string representation
func (c *FakeClient) String() string {
	return "FakeClient"
}

// Get looks up a cluster by name
func (c *FakeClient) Get(name string) (*cluster.Cluster, error) {
	return cluster.GetCluster(c, name)
}

// LabelCluster labels the given cluster
func (c *FakeClient) LabelCluster(name string, labels map[string]string) error {
	cluster, err := c.Get(name)
	if err != nil {
		return err
	}
	if cluster == nil {
		return fmt.Errorf("No cluster with name %s", name)
	}
	if cluster.Labels == nil {
		cluster.Labels = map[string]string{}
	}
	for k, v := range labels {
		cluster.Labels[k] = v
	}
	return nil
}
