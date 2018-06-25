package cmd

import (
	"io/ioutil"
	"testing"

	"path"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestValidateClusterDetails(t *testing.T) {
	o := CreateTerraformOptions{
		Flags: Flags{Cluster: []string{"foo=gke", "bar=gke"}},
	}
	err := o.validateClusterDetails()
	assert.NoError(t, err)
}

func TestValidateClusterDetailsFail(t *testing.T) {
	o := CreateTerraformOptions{
		Flags: Flags{Cluster: []string{"foo=gke", "bar=aks"}},
	}
	err := o.validateClusterDetails()
	assert.Error(t, err)
}

func TestCreateOrganisationFolderStructures(t *testing.T) {

	dir, err := ioutil.TempDir("", "test-create-org-struct")
	assert.NoError(t, err)

	c1 := Cluster{
		Name:     "foo",
		Provider: "gke",
	}
	c2 := Cluster{
		Name:     "bar",
		Provider: "gke",
	}

	o := CreateTerraformOptions{
		Clusters: []Cluster{c1, c2},
	}
	o.createOrganisationFolderStructure(dir)

	testDir1 := path.Join(dir, "clusters", "foo")
	exists, err := util.FileExists(testDir1)
	assert.NoError(t, err)
	assert.True(t, exists)
	testDir1NoGit := path.Join(testDir1, ".git")
	exists, err = util.FileExists(testDir1NoGit)
	assert.NoError(t, err)
	assert.False(t, exists)

	testDir2 := path.Join(dir, "clusters", "bar")
	exists, err = util.FileExists(testDir2)
	assert.NoError(t, err)
	assert.True(t, exists)

	testFile, err := util.LoadBytes(testDir1, "main.tf")
	assert.NotEmpty(t, testFile, "no terraform files found")
}
