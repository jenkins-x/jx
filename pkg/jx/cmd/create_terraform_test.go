package cmd

import (
	"io/ioutil"
	"testing"

	"path"

	"path/filepath"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
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

	c1 := GKECluster{
		_Name:     "foo",
		_Provider: "gke",
	}

	c2 := GKECluster{
		_Name:     "bar",
		_Provider: "gke",
	}

	clusterArray := []Cluster{c1, c2}

	for _, c := range clusterArray {
		assert.NotNil(t, c)
		_, ok := c.(GKECluster)
		assert.True(t, ok, "TEST: Unable to assert type to GKECluster")

		_, ok = c.(*GKECluster)
		assert.False(t, ok, "TEST: Unable to assert type to *GKECluster")
	}

	o := CreateTerraformOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				BatchMode: true,
			},
		},
		Clusters: clusterArray,
		Flags: Flags{
			OrganisationName:  "my-org",
			GKEProjectId:      "gke_project",
			GKESkipEnableApis: true,
			GKEZone:           "gke_zone",
			GKEMachineType:    "n1-standard-1",
			GKEMinNumOfNodes:  "3",
			GKEMaxNumOfNodes:  "5",
			GKEDiskSize:       "100",
			GKEAutoRepair:     true,
			GKEAutoUpgrade:    false,
		},
	}

	t.Logf("Creating org structure in %s", dir)

	clusterDefinitions, err := o.createOrganisationFolderStructure(dir)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(clusterDefinitions))

	t.Logf("Generated cluster definitions %s", clusterDefinitions)

	testDir1 := path.Join(dir, "clusters", "foo", "terraform")
	exists, err := util.FileExists(testDir1)
	assert.NoError(t, err)
	assert.True(t, exists, "Directory clusters/foo/terraform should exist")

	testDir1NoGit := path.Join(testDir1, ".git")
	exists, err = util.FileExists(testDir1NoGit)
	assert.NoError(t, err)
	assert.False(t, exists, "Directory clusters/foo/terraform/.git should not exist")

	testDir2 := path.Join(dir, "clusters", "bar", "terraform")
	exists, err = util.FileExists(testDir2)
	assert.NoError(t, err)
	assert.True(t, exists, "Directory clusters/bar/terraform should exist")

	gitignore, err := util.LoadBytes(dir, ".gitignore")
	assert.NotEmpty(t, gitignore, ".gitignore not found")

	testFile, err := util.LoadBytes(testDir1, "main.tf")
	assert.NotEmpty(t, testFile, "no terraform files found")
}

func TestCanCreateTerraformVarsFile(t *testing.T) {
	c := GKECluster{
		ProjectId:     "project",
		Zone:          "zone",
		MinNumOfNodes: "3",
		MaxNumOfNodes: "5",
		MachineType:   "n1-standard-2",
		DiskSize:      "100",
		AutoRepair:    true,
		AutoUpgrade:   false,
	}

	file, err := ioutil.TempFile("", "terraform-tf-vars")
	if err != nil {
		assert.Error(t, err)
	}

	path, err := filepath.Abs(file.Name())

	t.Logf("Writing output to %s", path)

	err = c.CreateTfVarsFile(path)
	if err != nil {
		assert.Error(t, err)
	}

	c2 := GKECluster{}
	c2.ParseTfVarsFile(path)

	assert.Equal(t, "project", c2.ProjectId)
	assert.Equal(t, "zone", c2.Zone)
	assert.Equal(t, "3", c2.MinNumOfNodes)
	assert.Equal(t, "5", c2.MaxNumOfNodes)
	assert.Equal(t, "n1-standard-2", c2.MachineType)
	assert.Equal(t, true, c2.AutoRepair)
	assert.Equal(t, false, c2.AutoUpgrade)

}

func TestCreateProwConfig(t *testing.T) {

	o := CreateTerraformOptions{

		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				kubeClient: fake.NewSimpleClientset(),
			},
		},
	}

	err := o.installProw("foo")
	assert.NoError(t, err)
}
