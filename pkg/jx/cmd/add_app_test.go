// +build integration

// TODO these are not supposed to be integration tests but there was a mistaken mis-usage of go-git which means they
//  randomly fail atm

package cmd_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/jenkins-x/jx/pkg/apps"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	uuid "github.com/satori/go.uuid"

	"k8s.io/helm/pkg/chartutil"

	"github.com/jenkins-x/jx/pkg/tests"

	"github.com/jenkins-x/jx/pkg/kube"

	"github.com/petergtz/pegomock"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ghodss/yaml"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/stretchr/testify/assert"

	google_protobuf "github.com/golang/protobuf/ptypes/any"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/cmd_test_helpers"
)

func TestAddAppForGitOps(t *testing.T) {
	t.Parallel()
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	repo := "http://chartmuseum.jenkins-x.io"
	description := "My test chart description"
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Alias:                alias,
		Repo:                 repo,
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:        name,
			Version:     version,
			Description: description,
		},
	}, testOptions.MockHelmer)
	o.Args = []string{name}
	err = o.Run()
	assert.NoError(t, err)
	pr, err := testOptions.FakeGitProvider.GetPullRequest(testOptions.OrgName, testOptions.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
	// Validate the PR has the right title, message
	assert.Equal(t, fmt.Sprintf("Add %s %s", name, version), pr.Title)
	assert.Equal(t, fmt.Sprintf("Add app %s %s", name, version), pr.Body)
	// Validate the branch name
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := testOptions.GetFullDevEnvDir(envDir)
	branchName, err := o.Git().Branch(devEnvDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("add-app-%s-%s", name, version), branchName)
	// Validate the updated Requirements.yaml
	requirements, err := helm.LoadRequirementsFile(filepath.Join(devEnvDir, helm.RequirementsFileName))
	assert.NoError(t, err)
	found := make([]*helm.Dependency, 0)
	for _, d := range requirements.Dependencies {
		if d.Name == name && d.Alias == alias {
			found = append(found, d)
		}
	}
	assert.Len(t, found, 1)
	assert.Equal(t, version, found[0].Version)
	app := &jenkinsv1.App{}
	appBytes, err := ioutil.ReadFile(filepath.Join(devEnvDir, name, "templates", name+"-app.yaml"))
	_ = yaml.Unmarshal(appBytes, app)
	assert.Equal(t, name, app.Labels[helm.LabelAppName])
	assert.Equal(t, version, app.Labels[helm.LabelAppVersion])
	assert.Equal(t, repo, app.Annotations[helm.AnnotationAppRepository])
	assert.Equal(t, description, app.Annotations[helm.AnnotationAppDescription])

}

func TestAddAppWithSecrets(t *testing.T) {
	// TODO enable this test again when is passing
	t.SkipNow()

	tests.SkipForWindows(t, "go-expect does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	// Needs console input to create secrets
	console := tests.NewTerminal(t)
	testOptions.CommonOptions.In = console.In
	testOptions.CommonOptions.Out = console.Out
	testOptions.CommonOptions.Err = console.Err

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	o.BatchMode = false

	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		Files: []*google_protobuf.Any{
			{
				TypeUrl: "values.schema.json",
				Value: []byte(`{
  "$id": "https:/jenkins-x.io/tests/basicTypes.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "description": "test values.yaml",
  "type": "object",
  "properties": {
    "tokenValue": {
      "type": "string",
      "format": "token"
    }
  }
}`),
			},
		},
	}, testOptions.MockHelmer)

	// Test interactive IO
	donec := make(chan struct{})
	// TODO Answer questions
	go func() {
		defer close(donec)
		// Test boolean type
		console.ExpectString("Enter a value for tokenValue")
		console.SendLine("abc")
		console.ExpectString(" ***")
		console.ExpectEOF()
	}()

	pegomock.When(testOptions.MockHelmer.UpgradeChart(
		pegomock.AnyString(),
		pegomock.EqString(name),
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
			assert.Len(t, valuesFiles, 2)
			_, valuesFileName := filepath.Split(valuesFiles[0])
			assert.Contains(t, valuesFileName, "values.yaml")
			bytes, err := ioutil.ReadFile(valuesFiles[0])
			assert.NoError(t, err)
			assert.Equal(t, `tokenValue:
  kind: Secret
  name: tokenvalue-secret
`, string(bytes))
			_, secretsFileName := filepath.Split(valuesFiles[1])
			assert.Contains(t, secretsFileName, "generatedSecrets.yaml")
			bytes, err = ioutil.ReadFile(valuesFiles[1])
			assert.NoError(t, err)
			assert.Equal(t, `appsGeneratedSecrets:
- Name: tokenvalue-secret
  key: token
  value: abc
`, string(bytes))
			// Check the template is in place
			_, err = os.Stat(filepath.Join(chart, "templates", "app-generated-secret-template.yaml"))
			assert.NoError(t, err)
			return []pegomock.ReturnValue{
				nil,
			}
		})

	err = o.Run()
	assert.NoError(t, err)
	err = console.Close()
	<-donec
	assert.NoError(t, err)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))

	// Validate that the secret reference is generated and the secret is in the chart
	// chart, _, _, _, _, _, _, _, _, valueFiles, _, _, _ :=
	testOptions.MockHelmer.VerifyWasCalledOnce().
		UpgradeChart(
			pegomock.AnyString(),
			pegomock.EqString(name),
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

func TestAddAppWithDefaults(t *testing.T) {

	tests.SkipForWindows(t, "go-expect does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	// Needs console input to create secrets
	console := tests.NewTerminal(t)
	testOptions.CommonOptions.In = console.In
	testOptions.CommonOptions.Out = console.Out
	testOptions.CommonOptions.Err = console.Err

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}

	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		Files: []*google_protobuf.Any{
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
      "default": "testing"
    }
  }
}`),
			},
		},
	}, testOptions.MockHelmer)

	// Test interactive IO
	donec := make(chan struct{})
	// TODO Answer questions
	go func() {
		defer close(donec)
		// Test boolean type
		console.ExpectString("Enter a value for name testing [Automatically accepted default value]")
		console.ExpectEOF()
	}()

	pegomock.When(testOptions.MockHelmer.UpgradeChart(
		pegomock.AnyString(),
		pegomock.EqString(name),
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
			assert.Equal(t, `name: testing
`, string(bytes))

			return []pegomock.ReturnValue{
				nil,
			}
		})

	err = o.Run()
	assert.NoError(t, err)
	err = console.Close()
	<-donec
	assert.NoError(t, err)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))

	testOptions.MockHelmer.VerifyWasCalledOnce().
		UpgradeChart(
			pegomock.AnyString(),
			pegomock.EqString(name),
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

func TestStashValues(t *testing.T) {
	namespace := "jx"

	tests.SkipForWindows(t, "go-expect does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	// Needs console input to create secrets
	console := tests.NewTerminal(t)
	testOptions.CommonOptions.In = console.In
	testOptions.CommonOptions.Out = console.Out
	testOptions.CommonOptions.Err = console.Err

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
		Namespace:            namespace,
	}
	o.Args = []string{name}

	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		Files: []*google_protobuf.Any{
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
      "default": "testing"
    }
  }
}`),
			},
		},
	}, testOptions.MockHelmer)

	err = o.Run()
	assert.NoError(t, err)
	appCRDName := fmt.Sprintf("%s-%s", name, name)
	jxClient, ns, err := testOptions.CommonOptions.JXClientAndDevNamespace()
	assert.NoError(t, err)
	appList, err := jxClient.JenkinsV1().Apps(ns).List(metav1.ListOptions{})
	assert.Equal(t, namespace, ns)
	assert.NoError(t, err)
	assert.Len(t, appList.Items, 1)
	app, err := jxClient.JenkinsV1().Apps(ns).Get(appCRDName, metav1.GetOptions{})
	assert.NoError(t, err)
	val, ok := app.Annotations[apps.ValuesAnnotation]
	assert.True(t, ok)
	dst, err := base64.StdEncoding.DecodeString(val)
	assert.NoError(t, err)
	assert.Equal(t, `{"name":"testing"}`, string(dst))

}

func TestAddAppForGitOpsWithSecrets(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	// Needs console input to create secrets
	console := tests.NewTerminal(t)
	testOptions.CommonOptions.In = console.In
	testOptions.CommonOptions.Out = console.Out
	testOptions.CommonOptions.Err = console.Err

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Alias:                alias,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	o.BatchMode = false

	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		Files: []*google_protobuf.Any{
			{
				TypeUrl: "values.schema.json",
				Value: []byte(`{
  "$id": "https:/jenkins-x.io/tests/basicTypes.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "description": "test values.yaml",
  "type": "object",
  "properties": {
    "tokenValue": {
      "type": "string",
      "format": "token"
    }
  }
}`),
			},
		},
	}, testOptions.MockHelmer)

	// Test interactive IO
	donec := make(chan struct{})
	// TODO Answer questions
	go func() {
		defer close(donec)
		// Test boolean type
		console.ExpectString("Enter a value for tokenValue")
		console.SendLine("abc")
		console.ExpectEOF()
	}()
	err = o.Run()
	assert.NoError(t, err)
	err = console.Close()
	<-donec
	assert.NoError(t, err)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))

	// Validate that the secret reference is generated
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	valuesFromPrPath := filepath.Join(testOptions.GetFullDevEnvDir(envDir), name, helm.ValuesFileName)
	_, err = os.Stat(valuesFromPrPath)
	assert.NoError(t, err)
	data, err := ioutil.ReadFile(valuesFromPrPath)
	assert.NoError(t, err)
	assert.Equal(t, `tokenValue:
  kind: Secret
  name: tokenvalue-secret
`, string(data))
	// Validate that vault has had the secret added
	path := strings.Join([]string{"gitOps", testOptions.OrgName, testOptions.DevEnvRepoInfo.Name, "tokenvalue-secret"},
		"/")
	value := map[string]interface{}{
		"token": "abc",
	}
	testOptions.MockVaultClient.VerifyWasCalledOnce().Write(path, value)
}

func TestAddApp(t *testing.T) {
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	// Can't run in parallel
	pegomock.RegisterMockTestingT(t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Repo:                 kube.DefaultChartMuseumURL,
		GitOps:               false,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	err = o.Run()
	assert.NoError(t, err)

	// Check chart was installed
	_, _, _, fetchDir, _, _, _ := testOptions.MockHelmer.VerifyWasCalledOnce().FetchChart(
		pegomock.EqString(name),
		pegomock.EqString(version),
		pegomock.AnyBool(),
		pegomock.AnyString(),
		pegomock.EqString(kube.DefaultChartMuseumURL),
		pegomock.AnyString(),
		pegomock.AnyString()).GetCapturedArguments()
	testOptions.MockHelmer.VerifyWasCalledOnce().
		UpgradeChart(
			pegomock.EqString(filepath.Join(fetchDir, name)),
			pegomock.EqString(name),
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

	// Verify the annotation
}

func TestAddAppFromPath(t *testing.T) {
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	// Can't run in parallel
	pegomock.RegisterMockTestingT(t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	commonOpts := *testOptions.CommonOptions

	// Make the local chart
	chartDir, err := ioutil.TempDir("", "local-chart")
	assert.NoError(t, err)
	chart := chart.Metadata{
		Version: version,
		Name:    name,
	}
	chartBytes, err := json.Marshal(chart)
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(chartDir, helm.ChartFileName), chartBytes, 0600)
	assert.NoError(t, err)

	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		GitOps:               false,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}

	o.Args = []string{chartDir}
	err = o.Run()
	assert.NoError(t, err)

	testOptions.MockHelmer.VerifyWasCalledOnce().
		UpgradeChart(
			pegomock.AnyString(),
			pegomock.EqString(name),
			pegomock.AnyString(),
			pegomock.EqString(version),
			pegomock.AnyBool(),
			pegomock.AnyInt(),
			pegomock.AnyBool(),
			pegomock.AnyBool(),
			pegomock.AnyStringSlice(),
			pegomock.AnyStringSlice(),
			pegomock.AnyString(),
			pegomock.AnyString(),
			pegomock.AnyString())

	// Verify the annotation
}

func TestAddLatestApp(t *testing.T) {

	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	// Can't run in parallel
	pegomock.RegisterMockTestingT(t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.1.1"
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Repo:                 kube.DefaultChartMuseumURL,
		GitOps:               false,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
	}, testOptions.MockHelmer)
	err = o.Run()
	assert.NoError(t, err)

	_, _, _, fetchDir, _, _, _ := testOptions.MockHelmer.VerifyWasCalledOnce().FetchChart(
		pegomock.EqString(name),
		pegomock.AnyString(),
		pegomock.AnyBool(),
		pegomock.AnyString(),
		pegomock.EqString(kube.DefaultChartMuseumURL),
		pegomock.AnyString(),
		pegomock.AnyString()).GetCapturedArguments()
	testOptions.MockHelmer.VerifyWasCalledOnce().
		UpgradeChart(
			pegomock.EqString(filepath.Join(fetchDir, name)),
			pegomock.EqString(name),
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

func TestAddAppWithValuesFileForGitOps(t *testing.T) {
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	values := map[string]interface{}{
		"cheese": "cheddar",
	}
	file, err := ioutil.TempFile("", "values.yaml")
	assert.NoError(t, err)
	data, err := yaml.Marshal(values)
	assert.NoError(t, err)
	_, err = file.Write(data)
	assert.NoError(t, err)

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Alias:                alias,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
		ValuesFiles:          []string{file.Name()},
	}
	o.Args = []string{name}
	err = o.Run()
	assert.NoError(t, err)
	// Validate that the values.yaml file is in the right place
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := testOptions.GetFullDevEnvDir(envDir)
	valuesFromPrPath := filepath.Join(devEnvDir, name, helm.ValuesFileName)
	_, err = os.Stat(valuesFromPrPath)
	assert.NoError(t, err)
	valuesFromPr := make(map[string]interface{})
	data, err = ioutil.ReadFile(valuesFromPrPath)
	assert.NoError(t, err)
	err = yaml.Unmarshal(data, &valuesFromPr)
	assert.NoError(t, err)
	assert.Equal(t, values, valuesFromPr)
}

func TestAddAppWithReadmeForGitOps(t *testing.T) {
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	description := "Example description"
	gitRepository := "https://git.fake/myorg/myrepo"
	releaseNotes := "https://issues.fake/myorg/myrepo/releasenotes/v0.0.1"
	release := jenkinsv1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", name, version),
		},
		Spec: jenkinsv1.ReleaseSpec{
			ReleaseNotesURL: releaseNotes,
			GitHTTPURL:      gitRepository,
		},
	}
	data, err := yaml.Marshal(release)
	assert.NoError(t, err)
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Alias:                alias,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:        name,
			Version:     version,
			Description: description,
		},
		Templates: []*chart.Template{
			{
				Name: "release.yaml",
				Data: data,
			},
		},
	}, testOptions.MockHelmer)
	err = o.Run()
	assert.NoError(t, err)
	// Validate that the README.md file is in the right place
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := testOptions.GetFullDevEnvDir(envDir)
	readmeFromPrPath := filepath.Join(devEnvDir, name, "README.MD")
	_, err = os.Stat(readmeFromPrPath)
	assert.NoError(t, err)
	data, err = ioutil.ReadFile(readmeFromPrPath)
	assert.NoError(t, err)
	readmeFromPr := string(data)
	assert.Equal(t, fmt.Sprintf(`# %s

|App Metadata|---|
| **Version** | %s |
| **Description** | %s |
| **Chart Repository** | %s |
| **Git Repository** | %s |
| **Release Notes** | %s |
`, name, version, description, kube.DefaultChartMuseumURL, gitRepository, releaseNotes), readmeFromPr)
	// Validate that the README.md file is in the right place
	releaseyamlFromPrPath := filepath.Join(devEnvDir, name, "release.yaml")
	_, err = os.Stat(releaseyamlFromPrPath)
	assert.NoError(t, err)
	data, err = ioutil.ReadFile(releaseyamlFromPrPath)
	assert.NoError(t, err)
	releaseFromPr := jenkinsv1.Release{}
	err = yaml.Unmarshal(data, &releaseFromPr)
	assert.NoError(t, err)
	assert.Equal(t, release, releaseFromPr)
}

func TestAddAppWithCustomReadmeForGitOps(t *testing.T) {
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Version:              version,
		Alias:                alias,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Verbose = true
	o.Args = []string{name}
	readmeFileName := "README.MD"
	readme := "Tasty Cheese!\n"
	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		Files: []*google_protobuf.Any{
			{
				TypeUrl: readmeFileName,
				Value:   []byte(readme),
			},
		},
	}, testOptions.MockHelmer)
	err = o.Run()
	assert.NoError(t, err)
	// Validate that the README.md file is in the right place
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := testOptions.GetFullDevEnvDir(envDir)
	readmeFromPrPath := filepath.Join(devEnvDir, name, readmeFileName)
	_, err = os.Stat(readmeFromPrPath)
	assert.NoError(t, err)
	data, err := ioutil.ReadFile(readmeFromPrPath)
	assert.NoError(t, err)
	readmeFromPr := string(data)
	assert.Equal(t, fmt.Sprintf(`# %s

|App Metadata|---|
| **Version** | %s |
| **Chart Repository** | %s |

## App README.MD

%s
`, name, version, kube.DefaultChartMuseumURL, readme), readmeFromPr)
}

func TestAddLatestAppForGitOps(t *testing.T) {
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	nameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name := nameUUID.String()
	version := "0.1.8"
	alias := fmt.Sprintf("%s-alias", name)
	commonOpts := *testOptions.CommonOptions
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: &commonOpts,
		},
		Alias:                alias,
		Repo:                 kube.DefaultChartMuseumURL,
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	o.Verbose = true

	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
	}, testOptions.MockHelmer)

	err = o.Run()
	assert.NoError(t, err)
	pr, err := testOptions.FakeGitProvider.GetPullRequest(testOptions.OrgName, testOptions.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
	// Validate the PR has the right title, message
	assert.Equal(t, fmt.Sprintf("Add %s %s", name, version), pr.Title)
	assert.Equal(t, fmt.Sprintf("Add app %s %s", name, version), pr.Body)
	// Validate the branch name
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := testOptions.GetFullDevEnvDir(envDir)
	branchName, err := o.Git().Branch(devEnvDir)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("add-app-%s-%s", name, version), branchName)
	// Validate the updated Requirements.yaml
	requirements, err := helm.LoadRequirementsFile(filepath.Join(devEnvDir, helm.RequirementsFileName))
	assert.NoError(t, err)
	found := make([]*helm.Dependency, 0)
	for _, d := range requirements.Dependencies {
		if d.Name == name && d.Alias == alias {
			found = append(found, d)
		}
	}
	assert.Len(t, found, 1)
	assert.Equal(t, version, found[0].Version)
}
