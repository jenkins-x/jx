// +build integration

package cmd_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/kube/quickstarts"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
)

func TestCreateQuickstartProjects(t *testing.T) {
	testDir, err := ioutil.TempDir("", "test-create-quickstart")
	assert.NoError(t, err)

	appName := "myvets"

	o := &cmd.CreateQuickstartOptions{
		GitHubOrganisations: []string{"petclinic-gcp"},
		Filter: quickstarts.QuickstartFilter{
			Text:        "petclinic-gcp/spring-petclinic-vets-service",
			ProjectName: appName,
		},
	}
	cmd.ConfigureTestOptions(&o.CommonOptions, gits.NewGitCLI(), helm.NewHelmCLI("helm", helm.V2, testDir, true))
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
		tests.AssertFileExists(t, jenkinsfile)
		tests.AssertFileExists(t, filepath.Join(appDir, "Dockerfile"))
		tests.AssertFileExists(t, filepath.Join(appDir, "charts", appName, "Chart.yaml"))
		tests.AssertFileExists(t, filepath.Join(appDir, "charts", appName, "Makefile"))
		tests.AssertFileDoesNotExist(t, filepath.Join(appDir, "charts", "spring-petclinic-vets-service", "Chart.yaml"))
	}
}
