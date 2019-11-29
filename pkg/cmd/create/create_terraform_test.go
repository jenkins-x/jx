// +build unit

package create_test

import (
	"errors"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/create"

	"github.com/stretchr/testify/assert"
)

func TestSetName(t *testing.T) {
	t.Parallel()
	c := create.GKECluster{}
	c.SetName("foo")
	assert.Equal(t, "foo", c.Name())
}

func TestSetProvider(t *testing.T) {
	t.Parallel()
	c := create.GKECluster{}
	c.SetProvider("bar")
	assert.Equal(t, "bar", c.Provider())
}

func TestValidateClusterDetails(t *testing.T) {
	t.Parallel()
	o := create.CreateTerraformOptions{
		Flags: create.Flags{Cluster: []string{"foo=gke", "bar=gke"}},
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
	o := create.CreateTerraformOptions{
		Flags: create.Flags{Cluster: []string{"foo=jx-infra"}},
	}
	err := o.ValidateClusterDetails()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(o.Clusters))
	assert.Equal(t, "foo", o.Clusters[0].Name())
	assert.Equal(t, "jx-infra", o.Clusters[0].Provider())
}

func TestValidateClusterDetailsFail(t *testing.T) {
	t.Parallel()
	o := create.CreateTerraformOptions{
		Flags: create.Flags{Cluster: []string{"foo=gke", "bar=aks"}},
	}
	err := o.ValidateClusterDetails()
	assert.EqualError(t, err, "invalid cluster provider type bar=aks, must be one of [gke jx-infra]")
	assert.Equal(t, 1, len(o.Clusters))
	assert.Equal(t, "foo", o.Clusters[0].Name())
	assert.Equal(t, "gke", o.Clusters[0].Provider())

	o = create.CreateTerraformOptions{
		Flags: create.Flags{ClusterName: "baz", CloudProvider: "gke"},
	}
	err = o.ValidateClusterDetails()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(o.Clusters))
	assert.Equal(t, "baz", o.Clusters[0].Name())
	assert.Equal(t, "gke", o.Clusters[0].Provider())
}

func TestValidateClusterDetailsForInvalidParameterCombination(t *testing.T) {
	t.Parallel()

	tests := []create.Flags{
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
		o := create.CreateTerraformOptions{Flags: flags}
		err := o.ValidateClusterDetails()
		assert.EqualError(t, err, "--cluster cannot be used in conjunction with --cluster-name or --cloud-provider")
	}
}

func TestValidateClusterArgs(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name                string
		org                 string
		shortClusterName    string
		expectedClusterName string
		err                 error
	}{
		{
			name:                "basic_cluster_name",
			org:                 "org",
			shortClusterName:    "dev",
			expectedClusterName: "org-dev",
		},
		{
			name:             "too_long_cluster_name",
			org:              "thisisareallylongorgname",
			shortClusterName: "thisisareallylongclustername",
			err:              errors.New("cluster name must not be longer than 27 characters - thisisareallylongorgname-thisisareallylongclustername"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := create.GKECluster{
				Organisation: tt.org,
			}
			g.SetName(tt.shortClusterName)

			err := g.Validate()
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
