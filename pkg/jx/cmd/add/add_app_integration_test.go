// +build integration

package add_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/add"

	"github.com/docker/docker/builder/dockerfile/command"

	"github.com/jenkins-x/jx/pkg/gits"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/jenkins-x/jx/pkg/util"

	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/tests"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"

	"k8s.io/helm/pkg/chartutil"

	"github.com/petergtz/pegomock"

	"github.com/ghodss/yaml"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/protobuf/ptypes/any"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pborman/uuid"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

var timeout = 5 * time.Second

func TestPreprocessSchema(t *testing.T) {
	t.SkipNow()
	pegomock.RegisterMockTestingT(t)
	helmer := helm_test.NewMockHelmer()
	appName := uuid.New()
	version := "1.0.0"

	appResource := jenkinsv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
		},
		Spec: jenkinsv1.AppSpec{
			SchemaPreprocessor: &corev1.Container{
				Image:   "gcr.io/jenkinsxio/builder-go:0.1.332",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					"kubectl get configmap $VALUES_SCHEMA_JSON_CONFIG_MAP_NAME -o yaml | sed 's/abc/def/' | kubectl" +
						" replace -f -",
				},
			},
		},
	}
	appResourceBytes, err := yaml.Marshal(appResource)
	assert.NoError(t, err)
	chartToCreate := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    appName,
			Version: version,
		},
		Files: []*any.Any{
			{
				TypeUrl: "values.schema.json",
				Value: []byte(`{
  "$id": "https:/jenkins-x.io/tests/basicTypes.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "description": "test values.yaml",
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "default": "abc"
    }
  }
}`),
			},
		},
		Templates: []*chart.Template{
			{
				Name: "app.yaml",
				Data: appResourceBytes,
			},
		},
	}
	helm_test.StubFetchChart(appName, "", kube.DefaultChartMuseumURL, chartToCreate, helmer)
	commonOpts := &opts.CommonOptions{
		BatchMode: true,
	}
	_, devEnv := commonOpts.GetDevEnv()
	var repoDir string
	configGitFn := func(dir string, gitInfo *gits.GitRepository, gitter gits.Gitter) error {
		repoDir = dir
		return nil
	}
	o := add.AddAppOptions{
		HelmUpdate: true,
		AddOptions: add.AddOptions{
			CommonOptions: commonOpts,
		},
		ConfigureGitCallback: configGitFn,
	}

	// Needs console output
	console := tests.NewTerminal(t, &timeout)
	defer console.Cleanup()
	o.In = console.In
	o.Out = console.Out
	o.Err = console.Err

	factory := o.GetFactory()
	if factory == nil {
		o.SetFactory(clients.NewFactory())
	}

	pegomock.When(helmer.UpgradeChart(

		pegomock.AnyString(),
		pegomock.EqString(fmt.Sprintf("jx-%s", appName)),
		pegomock.AnyString(),
		pegomock.EqString(version),
		pegomock.AnyBool(),
		pegomock.AnyInt(),
		pegomock.AnyBool(),
		pegomock.AnyBool(),
		pegomock.AnyStringSlice(),
		pegomock.AnyStringSlice(),
		pegomock.EqString(kube.DefaultChartMuseumURL),
		pegomock.AnyString(),
		pegomock.AnyString())).
		Then(func(params []pegomock.Param) pegomock.ReturnValues {
			// These assertion must happen inside the UpgradeChart function otherwise the chart dir will have been
			// deleted
			assert.IsType(t, "", params[0])
			assert.IsType(t, make([]string, 0), params[9])
			chart := params[0].(string)
			valuesFiles := params[9].([]string)
			isChartDir, err := chartutil.IsChartDir(chart)
			assert.NoError(t, err)
			assert.True(t, isChartDir)
			assert.Len(t, valuesFiles, 1)
			_, valuesFileName := filepath.Split(valuesFiles[0])
			assert.Contains(t, valuesFileName, "values.yaml")
			bytes, err := ioutil.ReadFile(valuesFiles[0])
			assert.NoError(t, err)
			assert.Equal(t, `name: def
`, string(bytes))

			return []pegomock.ReturnValue{
				nil,
			}
		})
	o.Args = []string{appName}
	realHelmer := o.Helm()
	// We do really need to call template!
	pegomock.When(helmer.Template(pegomock.AnyString(), pegomock.AnyString(), pegomock.AnyString(),
		pegomock.AnyString(), pegomock.AnyBool(), pegomock.AnyStringSlice(),
		pegomock.AnyStringSlice())).Then(func(params []pegomock.Param) pegomock.ReturnValues {
		chart, err := util.AsString(params[0])
		assert.NoError(t, err)
		releaseName, err := util.AsString(params[1])
		assert.NoError(t, err)
		ns, err := util.AsString(params[2])
		assert.NoError(t, err)
		outDir, err := util.AsString(params[3])
		assert.NoError(t, err)
		upgrade, err := util.AsBool(params[4])
		assert.NoError(t, err)
		values, err := util.AsSliceOfStrings(params[5])
		assert.NoError(t, err)
		valueFiles, err := util.AsSliceOfStrings(params[6])
		assert.NoError(t, err)
		err = realHelmer.Template(chart, releaseName, ns, outDir, upgrade, values, valueFiles)
		return pegomock.ReturnValues{
			err,
		}
	})
	o.SetHelm(helmer)
	o.Namespace = "jx"
	err = o.Run()
	assert.NoError(t, err)
	if o.GitOps {
		if repoDir == "" {
			envDir, err := o.CommonOptions.EnvironmentsDir()
			assert.NoError(t, err)
			gitRepo, err := gits.ParseGitURL(devEnv.Spec.Source.URL)
			assert.NoError(t, err)
			repoDir = GetFullDevEnvDir(envDir, gitRepo.Organisation, gitRepo.Name)
		}
		valuesFromPrPath := filepath.Join(repoDir, command.Env, appName, helm.ValuesFileName)
		_, err = os.Stat(valuesFromPrPath)
		assert.NoError(t, err)
		data, err := ioutil.ReadFile(valuesFromPrPath)
		assert.NoError(t, err)
		assert.Equal(t, `name: def
`, string(data))
	} else {
		helmer.VerifyWasCalledOnce().UpgradeChart(
			pegomock.AnyString(),
			pegomock.EqString(fmt.Sprintf("jx-%s", appName)),
			pegomock.AnyString(),
			pegomock.EqString(version),
			pegomock.AnyBool(),
			pegomock.AnyInt(),
			pegomock.AnyBool(),
			pegomock.AnyBool(),
			pegomock.AnyStringSlice(),
			pegomock.AnyStringSlice(),
			pegomock.EqString(kube.DefaultChartMuseumURL),
			pegomock.AnyString(),
			pegomock.AnyString())
	}

}

// GetFullDevEnvDir returns a dev environment including org name and env
func GetFullDevEnvDir(envDir, orgName string, repoName string) string {
	return filepath.Join(envDir, orgName, repoName)

}
