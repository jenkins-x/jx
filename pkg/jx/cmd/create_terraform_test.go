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
}

func TestValidateClusterDetailsForJxInfra(t *testing.T) {
	t.Parallel()
	o := cmd.CreateTerraformOptions{
		Flags: cmd.Flags{Cluster: []string{"foo=jx-infra"}},
	}
	err := o.ValidateClusterDetails()
	assert.NoError(t, err)
}

func TestValidateClusterDetailsFail(t *testing.T) {
	t.Parallel()
	o := cmd.CreateTerraformOptions{
		Flags: cmd.Flags{Cluster: []string{"foo=gke", "bar=aks"}},
	}
	err := o.ValidateClusterDetails()
	assert.Error(t, err)
}
