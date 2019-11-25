// +build integration

package create_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/create/options"

	"github.com/jenkins-x/jx/pkg/cmd/create"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestCreateOrganisationFolderStructures(t *testing.T) {
	originalJxHome, tempJxHome, err := testhelpers.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := testhelpers.CleanupTestJxHomeDir(originalJxHome, tempJxHome)
		assert.NoError(t, err)
	}()
	originalKubeCfg, tempKubeCfg, err := testhelpers.CreateTestKubeConfigDir()
	assert.NoError(t, err)
	defer func() {
		err := testhelpers.CleanupTestKubeConfigDir(originalKubeCfg, tempKubeCfg)
		assert.NoError(t, err)
	}()

	dir, err := ioutil.TempDir("", "test-create-org-struct")
	assert.NoError(t, err)

	c1 := &create.GKECluster{}
	c1.SetName("foo")
	c1.SetProvider("gke")

	c2 := &create.GKECluster{}
	c2.SetName("bar")
	c2.SetProvider("gke")

	clusterArray := []create.Cluster{c1, c2}

	for _, c := range clusterArray {
		assert.NotNil(t, c)
		_, ok := c.(*create.GKECluster)
		assert.True(t, ok, "TEST: Unable to assert type to GKECluster")
	}

	o := create.CreateTerraformOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: &opts.CommonOptions{
				BatchMode: true,
				In:        os.Stdin,
				Out:       os.Stdout,
				Err:       os.Stdout,
			},
		},
		Clusters: clusterArray,
		Flags: create.Flags{
			OrganisationName:  "my-org",
			GKEProjectID:      "gke_project",
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

	clusterDefinitions, err := o.CreateOrganisationFolderStructure(dir)
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
	c := create.GKECluster{
		ProjectID:     "project",
		Zone:          "zone",
		MinNumOfNodes: "3",
		MaxNumOfNodes: "5",
		MachineType:   "n1-standard-2",
		DiskSize:      "100",
		AutoRepair:    true,
		AutoUpgrade:   false,
		EnableVault:   true,
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

	c2 := create.GKECluster{}
	c2.ParseTfVarsFile(path)

	assert.Equal(t, "project", c2.ProjectID)
	assert.Equal(t, "zone", c2.Zone)
	assert.Equal(t, "3", c2.MinNumOfNodes)
	assert.Equal(t, "5", c2.MaxNumOfNodes)
	assert.Equal(t, "n1-standard-2", c2.MachineType)
	assert.Equal(t, true, c2.AutoRepair)
	assert.Equal(t, false, c2.AutoUpgrade)
	assert.Equal(t, true, c2.EnableVault)

}
