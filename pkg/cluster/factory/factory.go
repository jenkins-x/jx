package factory

import (
	"fmt"
	"os"

	"github.com/jenkins-x/jx/pkg/cluster"
	"github.com/jenkins-x/jx/pkg/cluster/gke"
)

// NewClientFromEnv uses environment variables to detect which kind of cluster we are running inside
func NewClientFromEnv() (cluster.Client, error) {
	if os.Getenv(cluster.EnvGKEProject) != "" && os.Getenv(cluster.EnvGKERegion) != "" {
		return gke.NewGKEFromEnv()
	}

	// lets try discover the current project
	return nil, fmt.Errorf("could not detect the cluter.Client from the environment variables")

}
