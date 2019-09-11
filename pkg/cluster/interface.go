package cluster

// Cluster represents a cluster
type Cluster struct {
	Name   string
	Labels map[string]string
	Status string
}

// Client represents a kubernetes cluster provider
type Client interface {
	// List lists the clusters in the current context - which is usually a project or user id etc
	List() ([]*Cluster, error)

	// ListFilter lists the clusters with the matching label filters
	ListFilter(strings map[string]string) ([]*Cluster, error)

	// Connect connects to the given cluster - returning an error if the connection cannot be made
	Connect(name string) error

	// String returns a text representation of the client
	String() string

	// LabelCluster adds labels to the given cluster
	LabelCluster(name string, labels map[string]string) error

	// Get looks up a given cluster by name returning nil if its not found
	Get(name string) (*Cluster, error)
}
