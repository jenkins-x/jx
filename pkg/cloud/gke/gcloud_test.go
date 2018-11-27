package gke

import (
	"github.com/jenkins-x/jx/pkg/kube/mocks"
	. "github.com/petergtz/pegomock"
	"k8s.io/client-go/tools/clientcmd/api"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRegionFromZone(t *testing.T) {
	t.Parallel()
	r := GetRegionFromZone("europe-west1-b")
	assert.Equal(t, r, "europe-west1")

	r = GetRegionFromZone("uswest1-d")
	assert.Equal(t, r, "uswest1")
}

func TestGetSimplifiedClusterName(t *testing.T) {
	t.Parallel()
	simpleName := GetSimplifiedClusterName("gke_jenkinsx-dev_europe-west1-b_my-cluster-name")

	assert.Equal(t, "my-cluster-name", simpleName)
}

func TestShortClusterName(t *testing.T) {
	t.Parallel()
	kuber := kube_test.NewMockKuber()

	config := api.Config{
		CurrentContext: "myContext",
		Contexts: map[string]*api.Context{
			"myContext": {Cluster: "short-cluster-name"},
		},
	}
	When(kuber.LoadConfig()).ThenReturn(&config, nil, nil)

	clusterName, err := ShortClusterName(kuber)

	assert.NoError(t, err)
	assert.Equal(t, "short", clusterName)
}

func TestClusterName(t *testing.T) {
	t.Parallel()
	kuber := kube_test.NewMockKuber()

	config := api.Config{
		CurrentContext: "myContext",
		Contexts: map[string]*api.Context{
			"myContext": {Cluster: "my-cluster-name"},
		},
	}
	When(kuber.LoadConfig()).ThenReturn(&config, nil, nil)

	clusterName, err := ClusterName(kuber)

	assert.NoError(t, err)
	assert.Equal(t, "my-cluster-name", clusterName)
}
