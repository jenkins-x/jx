package amazon

import (
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// AWSCli is an abstraction over the AWS CLI operations
type AWSCli struct{}

// ConnectToClusterWithAWSCLI will modify the kube-config file to add the provided cluster and change context to it
func (AWSCli) ConnectToClusterWithAWSCLI(clusterName string) error {
	args := []string{"eks", "update-kubeconfig", "--name", clusterName}

	cmd := util.Command{
		Name: "aws",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return errors.Wrapf(err, "failed to connect to region cluster %s", clusterName)
	}
	return nil
}
