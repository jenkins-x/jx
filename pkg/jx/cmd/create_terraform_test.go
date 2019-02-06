package cmd_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
)

func TestSetName(t *testing.T) {
	t.Parallel()
	c := cmd.GKECluster{}
	c.SetName("foo")
	assert.Equal(t, "foo", c.Name())
}

func TestSetProvider(t *testing.T) {
	t.Parallel()
	c := cmd.GKECluster{}
	c.SetProvider("bar")
	assert.Equal(t, "bar", c.Provider())
}

func TestValidateClusterDetails(t *testing.T) {
	t.Parallel()
	o := cmd.CreateTerraformOptions{
		Flags: cmd.Flags{Cluster: []string{"foo=gke", "bar=gke"}},
	}
	err := o.ValidateClusterDetails()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(o.Clusters))
	assert.Equal(t, "foo", o.Clusters[0].Name())
	assert.Equal(t, "gke", o.Clusters[0].Provider())
	assert.Equal(t, "bar", o.Clusters[1].Name())
	assert.Equal(t, "gke", o.Clusters[1].Provider())
}

func TestValidateClusterDetailsForJxInfra(t *testing.T) {
	t.Parallel()
	o := cmd.CreateTerraformOptions{
		Flags: cmd.Flags{Cluster: []string{"foo=jx-infra"}},
	}
	err := o.ValidateClusterDetails()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(o.Clusters))
	assert.Equal(t, "foo", o.Clusters[0].Name())
	assert.Equal(t, "jx-infra", o.Clusters[0].Provider())
}

func TestValidateClusterDetailsFail(t *testing.T) {
	t.Parallel()
	o := cmd.CreateTerraformOptions{
		Flags: cmd.Flags{Cluster: []string{"foo=gke", "bar=aks"}},
	}
	err := o.ValidateClusterDetails()
	assert.EqualError(t, err, "invalid cluster provider type bar=aks, must be one of [gke jx-infra]")
	assert.Equal(t, 1, len(o.Clusters))
	assert.Equal(t, "foo", o.Clusters[0].Name())
	assert.Equal(t, "gke", o.Clusters[0].Provider())

	o = cmd.CreateTerraformOptions{
		Flags: cmd.Flags{ClusterName: "baz", CloudProvider: "gke"},
	}
	err = o.ValidateClusterDetails()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(o.Clusters))
	assert.Equal(t, "baz", o.Clusters[0].Name())
	assert.Equal(t, "gke", o.Clusters[0].Provider())
}

func TestValidateClusterDetailsForInvalidParameterCombination(t *testing.T) {
	t.Parallel()

	tests := []cmd.Flags{
		{
			Cluster:     []string{"foo=jx-infra"},
			ClusterName: "foo",
		},
		{
			Cluster:       []string{"foo=jx-infra"},
			CloudProvider: "gke",
		},
	}

	for _, flags := range tests {
		o := cmd.CreateTerraformOptions{Flags: flags}
		err := o.ValidateClusterDetails()
		assert.EqualError(t, err, "--cluster cannot be used in conjunction with --cluster-name or --cloud-provider")
	}
}
