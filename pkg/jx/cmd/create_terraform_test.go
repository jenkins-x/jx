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
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				BatchMode: true,
			},
		},
		Clusters: []Cluster{c1, c2},
		Flags: Flags{
			OrganisationRepoName: "my-org",
			GKEProjectId:         "gke_project",
			GKEZone:              "gke_zone",
			GKEMachineType:       "n1-standard-1",
			GKEMinNumOfNodes:     "3",
			GKEMaxNumOfNodes:     "5",
			GKEDiskSize:          "100",
			GKEAutoRepair:        true,
			GKEAutoUpgrade:       false,
		},
	}

	o.createOrganisationFolderStructure(dir)

	testDir1 := path.Join(dir, "clusters", "foo", "terraform")
	exists, err := util.FileExists(testDir1)
	assert.NoError(t, err)
	assert.True(t, exists)
	testDir1NoGit := path.Join(testDir1, ".git")
	exists, err = util.FileExists(testDir1NoGit)
	assert.NoError(t, err)
	assert.False(t, exists)

	testDir2 := path.Join(dir, "clusters", "bar", "terraform")
	exists, err = util.FileExists(testDir2)
	assert.NoError(t, err)
	assert.True(t, exists)

	gitignore, err := util.LoadBytes(dir, ".gitignore")
	assert.NotEmpty(t, gitignore, ".gitignore not founc")

	testFile, err := util.LoadBytes(testDir1, "main.tf")
	assert.NotEmpty(t, testFile, "no terraform files found")
}
