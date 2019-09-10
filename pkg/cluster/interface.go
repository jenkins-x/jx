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
	List() ([]Cluster, error)

	// Connect connects to the given cluster - returning an error if the connection cannot be made
	Connect(name string) error

	// String returns a text representation of the client
	String() string
}
