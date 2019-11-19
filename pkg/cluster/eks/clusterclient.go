package eks

import (
	"fmt"

	awsAPI "github.com/aws/aws-sdk-go/aws"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/pkg/cluster"
)

// AWSClusterClient that will provide functions to interact with EKS through the AWS API or the AWS CLI
type AWSClusterClient struct {
	awsClusterClient *amazon.AWSCli
	awsEKSAPI        *amazon.EksClusterOptions
}

// NewAWSClusterClient will return an AWSClusterClient
func NewAWSClusterClient() (cluster.Client, error) {
	return &AWSClusterClient{
		awsClusterClient: &amazon.AWSCli{},
		awsEKSAPI:        &amazon.EksClusterOptions{},
	}, nil
}

// List will return a slice of every EKS cluster existing in the configured region
func (a AWSClusterClient) List() ([]*cluster.Cluster, error) {
	return a.awsEKSAPI.ListClusters()
}

// ListFilter lists the clusters with the matching label filters
func (a AWSClusterClient) ListFilter(tags map[string]string) ([]*cluster.Cluster, error) {
	return cluster.ListFilter(a, tags)
}

// Connect connects to the given cluster - returning an error if the connection cannot be made
func (a AWSClusterClient) Connect(cluster *cluster.Cluster) error {
	return a.awsClusterClient.ConnectToClusterWithAWSCLI(cluster.Name)
}

// String returns a text representation of the client
func (a AWSClusterClient) String() string {
	return fmt.Sprintf("EKS Cluster client")
}

// SetClusterLabels adds labels to the given cluster
func (a AWSClusterClient) SetClusterLabels(cluster *cluster.Cluster, clusterTags map[string]string) error {
	// AWS works with Tags, should be equal
	return a.awsEKSAPI.AddTagsToCluster(cluster.Name, awsAPI.StringMap(clusterTags))
}

// Get looks up a given cluster by name returning nil if its not found
func (a AWSClusterClient) Get(clusterName string) (*cluster.Cluster, error) {
	describedCluster, _, err := a.awsEKSAPI.DescribeCluster(clusterName)
	return describedCluster, err
}
