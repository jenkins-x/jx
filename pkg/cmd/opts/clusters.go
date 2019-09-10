package opts

import (
	gcp "github.com/jenkins-x/jx/pkg/cloud/gke"

	"github.com/jenkins-x/jx/pkg/cluster"
	"github.com/jenkins-x/jx/pkg/cluster/gke"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ClusterOptions used to determine which kind of cluster to query
type ClusterOptions struct {
	GKE GKEClusterOptions
}

// GKEClusterOptions GKE specific configurations
type GKEClusterOptions struct {
	Project string
	Region  string
}

func (o *ClusterOptions) CreateClient() (cluster.Client, error) {
	if o.GKE.Project != "" || o.GKE.Region != "" {
		if o.GKE.Project == "" {
			return nil, util.MissingOption("gke-project")
		}
		if o.GKE.Region == "" {
			return nil, util.MissingOption("gke-region")
		}
	}

	// lets try detect the GKE project...
	g := &gcp.GCloud{}
	project, err := g.CurrentProject()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to detect if we are using GKE")
	}
	o.GKE.Project = project
	return gke.NewGKE(project, o.GKE.Region)

	//return nil, fmt.Errorf("could not detect the kind of cluster via the command line options")
}

func (o *ClusterOptions) AddClusterFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.GKE.Project, "gke-project", "", "", "The GKE project name")
	cmd.Flags().StringVarP(&o.GKE.Region, "gke-region", "", "", "The GKE project name")
}
