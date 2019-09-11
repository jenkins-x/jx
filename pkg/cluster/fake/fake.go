package fake

import (
	"github.com/jenkins-x/jx/pkg/cluster"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
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
func (c *FakeClient) Connect(cluster *cluster.Cluster) error {
	log.Logger().Infof("fake cluster connecting to cluster: %s", util.ColorInfo(cluster.Name))
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

// SetClusterLabels labels the given cluster
func (c *FakeClient) SetClusterLabels(cluster *cluster.Cluster, labels map[string]string) error {
	cluster.Labels = labels
	return nil
}
