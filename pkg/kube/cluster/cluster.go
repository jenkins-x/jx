package cluster

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Name gets the cluster name from the current context
// Note that this just reads the ClusterName from the local kube config, which can be renamed (but is unlikely to happen)
func Name(kuber kube.Kuber) (string, error) {
	context, err := Context(kuber)
	if err != nil {
		return "", err
	}
	if context == nil {
		return "", errors.New("kube context was nil")
	}
	// context.Cluster will likely be in the form gke_<accountName>_<region>_<clustername>
	// Trim off the crud from the beginning context.Cluster
	return SimplifiedClusterName(context.Cluster), nil
}

// Context returns the current kube context
func Context(kuber kube.Kuber) (*api.Context, error) {
	config, _, err := kuber.LoadConfig()
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	return kube.CurrentContext(config), nil
}

// ShortName returns a short clusters name. Eg, if ClusterName would return tweetypie-jenkinsx-dev, ShortClusterName
// would return tweetypie. This is needed because GCP has character limits on things like service accounts (6-30 chars)
// and combining a long cluster name and a long vault name exceeds this limit
func ShortName(kuber kube.Kuber) (string, error) {
	clusterName, err := Name(kuber)
	if err != nil {
		return "", errors.Wrap(err, "retrieveing the cluster name")
	}
	end := len(clusterName) - 1
	if end > 16 {
		end = 16
	}
	shortClusterName := clusterName[0:end]
	if strings.HasSuffix(shortClusterName, "_") || strings.HasSuffix(shortClusterName, "-") {
		shortClusterName = shortClusterName[0 : end-1]
	}
	return shortClusterName, nil
}

// SimplifiedClusterName get the simplified cluster name from the long-winded context cluster name that gets generated
// GKE cluster names as defined in the kube config are of the form gke_<projectname>_<region>_<clustername>
// This method will return <clustername> in the above
func SimplifiedClusterName(complexClusterName string) string {
	split := strings.Split(complexClusterName, "_")
	return split[len(split)-1]
}
