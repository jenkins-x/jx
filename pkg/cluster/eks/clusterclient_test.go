package eks

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"github.com/stretchr/testify/assert"
)

type mockedEKS struct {
	eksiface.EKSAPI
	ListClustersResponses    []interface{}
	DescribeClusterResponses map[string]*eks.Cluster
}

func (m mockedEKS) ListClusters(input *eks.ListClustersInput) (*eks.ListClustersOutput, error) {
	clusters := m.ListClustersResponses[0].([]*string)
	var nextToken *string
	if m.ListClustersResponses[1] != nil {
		nextToken = m.ListClustersResponses[1].(*string)
	}
	return &eks.ListClustersOutput{
		Clusters:  clusters,
		NextToken: nextToken,
	}, nil
}

func (m mockedEKS) DescribeCluster(input *eks.DescribeClusterInput) (*eks.DescribeClusterOutput, error) {
	return &eks.DescribeClusterOutput{
		Cluster: m.DescribeClusterResponses[*input.Name],
	}, nil
}

func TestAWSClusterClient_List(t *testing.T) {
	commands, err := amazon.NewEKSClusterOptions(mockedEKS{
		ListClustersResponses: []interface{}{aws.StringSlice([]string{"cluster-1"}), nil},
		DescribeClusterResponses: map[string]*eks.Cluster{
			"cluster-1": {
				Arn:      aws.String("fake:arn:for:cluster"),
				Endpoint: aws.String("this.is.an/endpoint"),
				Name:     aws.String("cluster-1"),
				Status:   aws.String("ACTIVE"),
				Tags: aws.StringMap(map[string]string{
					"tag1": "value1",
				}),
			},
		},
	})
	assert.NoError(t, err)

	c := AWSClusterClient{
		awsClusterClient: &amazon.AWSCli{},
		awsEKSAPI:        commands,
	}

	clusters, err := c.List()
	assert.NoError(t, err)

	assert.Len(t, clusters, 1, "there must be at least one returned cluster from List")
	if len(clusters) > 0 {
		cluster := clusters[0]
		assert.Equal(t, "cluster-1", cluster.Name)
		assert.Equal(t, "ACTIVE", cluster.Status)
		assert.Equal(t, "this.is.an/endpoint", cluster.Location)
		assert.Equal(t, map[string]string{
			"tag1": "value1",
		}, cluster.Labels)
	}
}

func TestAWSClusterClient_Get(t *testing.T) {
	commands, err := amazon.NewEKSClusterOptions(mockedEKS{
		DescribeClusterResponses: map[string]*eks.Cluster{
			"cluster-1": {
				Arn:      aws.String("fake:arn:for:cluster"),
				Endpoint: aws.String("this.is.an/endpoint"),
				Name:     aws.String("cluster-1"),
				Status:   aws.String("ACTIVE"),
				Tags: aws.StringMap(map[string]string{
					"tag1": "value1",
				}),
			},
		},
	})
	assert.NoError(t, err)

	c := AWSClusterClient{
		awsClusterClient: &amazon.AWSCli{},
		awsEKSAPI:        commands,
	}

	obtainedCluster, err := c.Get("cluster-1")
	assert.NoError(t, err)

	assert.Equal(t, "cluster-1", obtainedCluster.Name)
	assert.Equal(t, "ACTIVE", obtainedCluster.Status)
	assert.Equal(t, "this.is.an/endpoint", obtainedCluster.Location)
	assert.Equal(t, map[string]string{
		"tag1": "value1",
	}, obtainedCluster.Labels)
}

func TestAWSClusterClient_ListFilter(t *testing.T) {
	commands, err := amazon.NewEKSClusterOptions(mockedEKS{
		ListClustersResponses: []interface{}{aws.StringSlice([]string{"cluster-1", "cluster-2", "cluster-3"}), nil},
		DescribeClusterResponses: map[string]*eks.Cluster{
			"cluster-1": {
				Arn:      aws.String("fake:arn:for:cluster"),
				Endpoint: aws.String("this.is.an/endpoint"),
				Name:     aws.String("cluster-1"),
				Status:   aws.String("ACTIVE"),
				Tags: aws.StringMap(map[string]string{
					"tag1": "value1",
				}),
			},
			"cluster-2": {
				Arn:      aws.String("fake:arn:for:cluster"),
				Endpoint: aws.String("this.is.an/endpoint"),
				Name:     aws.String("cluster-2"),
				Status:   aws.String("ACTIVE"),
				Tags: aws.StringMap(map[string]string{
					"IWANT": "THIS_CLUSTER",
				}),
			},
			"cluster-3": {
				Arn:      aws.String("fake:arn:for:cluster"),
				Endpoint: aws.String("this.is.an/endpoint"),
				Name:     aws.String("cluster-3"),
				Status:   aws.String("ACTIVE"),
				Tags: aws.StringMap(map[string]string{
					"tag1": "value1",
				}),
			},
		},
	})
	assert.NoError(t, err)

	c := AWSClusterClient{
		awsClusterClient: &amazon.AWSCli{},
		awsEKSAPI:        commands,
	}

	clusters, err := c.ListFilter(map[string]string{
		"IWANT": "THIS_CLUSTER",
	})
	assert.NoError(t, err)

	assert.Len(t, clusters, 1, "it must return only one cluster matching tags")
	if len(clusters) > 0 {
		obtainedCluster := clusters[0]
		assert.Equal(t, "cluster-2", obtainedCluster.Name)
	}
}
