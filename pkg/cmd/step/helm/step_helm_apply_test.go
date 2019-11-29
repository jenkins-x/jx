// +build unit

package helm_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/opts/step"

	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	helm_cmd "github.com/jenkins-x/jx/pkg/cmd/step/helm"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/mholt/archiver"
	"github.com/stretchr/testify/assert"
)

func TestApplyAppsTemplateOverrides(t *testing.T) {
	testOptions := testhelpers.CreateAppTestOptions(true, "dummy", t)
	_, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)

	envsDir, err := testOptions.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	absoluteRepoPath := filepath.Join(envsDir, testOptions.DevEnv.Name)

	//Create the charts folder
	chartsPath := filepath.Join(absoluteRepoPath, "charts")
	exists, err := util.DirExists(chartsPath)
	assert.NoError(t, err)
	if !exists {
		err = os.Mkdir(chartsPath, os.ModePerm)
		assert.NoError(t, err)
	}

	testFile := path.Join("test_data", "apply_env", "jx-app-dummy-0.0.3.tgz")
	_, err = os.Stat(testFile)
	assert.NoError(t, err)

	data, err := ioutil.ReadFile(testFile)
	assert.NoError(t, err)

	chartFilePath := filepath.Join(chartsPath, "jx-app-dummy-0.0.3.tgz")
	err = ioutil.WriteFile(chartFilePath, data, util.DefaultWritePermissions)
	assert.NoError(t, err)

	sto := helm_cmd.StepHelmApplyOptions{
		StepHelmOptions: helm_cmd.StepHelmOptions{
			Dir: absoluteRepoPath,
			StepOptions: step.StepOptions{
				CommonOptions: testOptions.CommonOptions,
			},
		},
		ReleaseName: "jx-app-dummy",
	}

	err = sto.Run()
	assert.NoError(t, err)

	_, err = os.Stat(chartFilePath)
	assert.NoError(t, err)
	uuid, _ := uuid.NewUUID()
	explodedFolderPath := filepath.Join(os.TempDir(), uuid.String())
	archiver.Unarchive(chartFilePath, explodedFolderPath)

	appsYamlFilePath := filepath.Join(explodedFolderPath, "jx-app-dummy", "templates", "app.yaml")
	chartData, err := ioutil.ReadFile(appsYamlFilePath)
	assert.NoError(t, err)

	app := v1.App{}
	err = yaml.Unmarshal(chartData, &app)
	assert.NoError(t, err)

	assert.Equal(t, "jx-app-dummy", app.Labels[helm.LabelAppName])

}
