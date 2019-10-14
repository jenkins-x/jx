package v1

import (
	"path/filepath"
)

// RemoteConnector specifies the namespace in the remote cluster
type RemoteConnector struct {
	GKE *GKEConnector `json:"gcp,omitempty" protobuf:"bytes,1,opt,name=gcp"`
}

// Key returns the key used for caching connectors
func (c *RemoteConnector) Path() string {
	if c.GKE != nil {
		return c.GKE.Path()
	}
	return "unknown"
}

// GKEConnector the connection details for using Google Cloud
type GKEConnector struct {
	Project string `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`
	Cluster string `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`
	Region  string `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`
	Zone    string `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`
}

func (c *GKEConnector) Path() string {
	if c.Region != "" {
		return filepath.Join("gcp", c.Project, c.Cluster, "region", c.Region)
	} else {
		return filepath.Join("gcp", c.Project, c.Cluster, "zone", c.Zone)
	}
}
