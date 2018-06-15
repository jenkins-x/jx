package cmd

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/quickstarts"
	"github.com/stretchr/testify/assert"
)

func TestCreateQuckstartProjects(t *testing.T) {
	testDir, err := ioutil.TempDir("", "test-create-quickstart")
	assert.NoError(t, err)

	appName := "myvets"

	o := &CreateQuickstartOptions{
		GitHubOrganisations: []string{"petclinic-gcp"},
		Filter: quickstarts.QuickstartFilter{
			Text:        "petclinic-gcp/spring-petclinic-vets-service",
			ProjectName: appName,
		},
	}
	configureTestOptions(&o.CommonOptions)
	o.Dir = testDir
	o.OutDir = testDir
	o.DryRun = true
	o.DisableMaven = true
	o.Verbose = true
	o.IgnoreTeam = true
	o.Repository = appName

	err = o.Run()
	assert.NoError(t, err)
	if err == nil {
		appDir := filepath.Join(testDir, appName)
		jenkinsfile := filepath.Join(appDir, "Jenkinsfile")
		assertFileExists(t, jenkinsfile)
		assertFileExists(t, filepath.Join(appDir, "Dockerfile"))
		assertFileExists(t, filepath.Join(appDir, "charts", appName, "Chart.yaml"))
		assertFileExists(t, filepath.Join(appDir, "charts", appName, "Makefile"))
		assertFileDoesNotExist(t, filepath.Join(appDir, "charts", "spring-petclinic-vets-service", "Chart.yaml"))
	}
}
