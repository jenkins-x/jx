package cmd_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	expect "github.com/Netflix/go-expect"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/acarl005/stripansi"
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/cmd_test_helpers"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/petergtz/pegomock"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
)

func TestGetAppsGitops(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, t)
	namespace := ""

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	envDir, err := testOptions.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := testOptions.GetFullDevEnvDir(envDir)
	_, devEnv := testOptions.CommonOptions.GetDevEnv()

	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace: namespace,
	}
	appResourceLocation := filepath.Join(devEnvDir, name1, "templates", "app.yaml")
	app := &v1.App{}
	appBytes, err := ioutil.ReadFile(appResourceLocation)
	err = yaml.Unmarshal(appBytes, app)
	app.Labels[helm.LabelReleaseName] = fmt.Sprintf("%s-%s", namespace, name1)
	app.Namespace = namespace
	cmd.ConfigureTestOptionsWithResources(getAppOptions.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			devEnv,
			app,
		},
		gits.NewGitLocal(),
		testOptions.FakeGitProvider,
		testOptions.MockHelmer,
		testOptions.CommonOptions.ResourcesInstaller(),
	)
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	getAppOptions.Args = []string{}
	err = getAppOptions.Run()
	assert.NoError(t, err)
	err = console.Close()

	assert.NoError(t, err)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))
	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, name1)
}

func TestGetApps(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	name2, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace: namespace,
	}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	getAppOptions.Args = []string{}
	err = getAppOptions.Run()
	assert.NoError(t, err)
	err = console.Close()

	assert.NoError(t, err)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))
	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, name1)
	assert.Contains(t, output, name2)
}

func TestGetAppsWithErrorGettingStatus(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	_, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	_, _, _, err = testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace: namespace,
	}

	pegomock.When(getAppOptions.Helm().ListReleases(pegomock.EqString(namespace))).
		ThenReturn(nil, nil, errors.New("this is an error"))

	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	getAppOptions.Args = []string{}
	err = getAppOptions.Run()
	assert.NoError(t, err)
	err = console.Close()

	assert.NoError(t, err)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, "Status")
	assert.NotContains(t, output, "DEPLOYED")
}

func TestGetAppsWithErrorGettingStatusWithOutput(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	_, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	_, _, _, err = testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
			Output:        "json",
		},
		Namespace: namespace,
	}

	pegomock.When(getAppOptions.Helm().ListReleases(pegomock.EqString(namespace))).
		ThenReturn(nil, nil, errors.New("this is an error"))

	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	getAppOptions.Args = []string{}
	err = getAppOptions.Run()
	assert.NoError(t, err)
	err = console.Close()

	assert.NoError(t, err)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, "\"status\":\"\"")
}

func TestGetApp(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	name2, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace: namespace,
	}
	getAppOptions.Args = []string{name1}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, name1)
	assert.NotContains(t, output, name2)
}

func TestGetAppWithShortName(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	name2, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace: namespace,
	}
	getAppOptions.Args = []string{name1}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, name1)
	assert.NotContains(t, output, name2)
}

func TestGetAppsHasStatus(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace: namespace,
	}
	formattedName := fmt.Sprintf("%s-%s", namespace, name1)
	pegomock.When(getAppOptions.Helm().ListReleases(pegomock.EqString(namespace))).
		ThenReturn(map[string]helm.ReleaseSummary{
			formattedName: {Status: "DEPLOYED"},
		}, []string{formattedName}, nil)

	getAppOptions.Args = []string{name1}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "DEPLOYED")
}

func TestGetAppsAsJson(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
			Output:        "json",
		},
		Namespace: namespace,
	}
	formattedName := fmt.Sprintf("%s-%s", namespace, name1)
	pegomock.When(getAppOptions.Helm().ListReleases(pegomock.EqString(namespace))).
		ThenReturn(map[string]helm.ReleaseSummary{
			formattedName: {Status: "DEPLOYED"},
		}, []string{formattedName}, nil)

	getAppOptions.Args = []string{name1}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	testDataPath := filepath.Join("test_data", "get_apps", "get_apps_table_json.json")
	_, err = os.Stat(testDataPath)
	assert.NoError(t, err)

	expectedJSON, _ := ioutil.ReadFile(testDataPath)
	expectedJSONString := strings.Replace(string(expectedJSON), "<NAME_PLACEHOLDER>", name1, 1)
	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.JSONEq(t, expectedJSONString, output)
}

func TestGetAppsAsYaml(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
			Output:        "yaml",
		},
		Namespace: namespace,
	}
	formattedName := fmt.Sprintf("%s-%s", namespace, name1)
	pegomock.When(getAppOptions.Helm().ListReleases(pegomock.EqString(namespace))).
		ThenReturn(map[string]helm.ReleaseSummary{
			formattedName: {Status: "DEPLOYED"},
		}, []string{formattedName}, nil)

	getAppOptions.Args = []string{name1}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	testDataPath := filepath.Join("test_data", "get_apps", "get_apps_table_yaml.yaml")
	_, err = os.Stat(testDataPath)
	assert.NoError(t, err)

	expectedYaml, _ := ioutil.ReadFile(testDataPath)
	expectedYamlString := strings.Replace(stripansi.Strip(string(expectedYaml)), "<NAME_PLACEHOLDER>", name1, 1)
	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Equal(t, expectedYamlString, output)
}

func TestGetAppsResourcesStatusTooManyApps(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	name2, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace:  namespace,
		ShowStatus: true,
	}
	getAppOptions.Args = []string{name1, name2}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	consoleOutput := stripansi.Strip(string(outBytes))
	assert.Equal(t, "Different apps provided. Provide only one app to check the status of\n", consoleOutput)
}

func TestGetAppsResourcesStatusJsonFormat(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
			Output:        "json",
		},
		Namespace:  namespace,
		ShowStatus: true,
	}
	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	testDataPath := filepath.Join("test_data", "get_apps", "get_app_status.json")
	_, err = os.Stat(testDataPath)
	assert.NoError(t, err)
	expectedJSON, _ := ioutil.ReadFile(testDataPath)
	pegomock.When(getAppOptions.Helm().StatusReleaseWithOutput(pegomock.EqString(namespace),
		pegomock.AnyString(), pegomock.EqString("json"))).
		ThenReturn(string(expectedJSON), nil)
	getAppOptions.Args = []string{name1}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	h := cmd.HelmOutput{}
	err = json.Unmarshal(outBytes, &h)
	assert.NoErrorf(t, err, "Error parsing json")
	assert.NotEmpty(t, h.Resources)
}

func TestGetAppsResourcesStatusYamlFormat(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
			Output:        "yaml",
		},
		Namespace:  namespace,
		ShowStatus: true,
	}

	testDataPath := filepath.Join("test_data", "get_apps", "get_app_status.json")
	_, err = os.Stat(testDataPath)
	assert.NoError(t, err)
	expectedJSON, _ := ioutil.ReadFile(testDataPath)

	pegomock.When(testOptions.MockHelmer.StatusReleaseWithOutput(pegomock.EqString(namespace),
		pegomock.AnyString(), pegomock.EqString("json"))).
		ThenReturn(string(expectedJSON), nil)
	getAppOptions.Args = []string{name1}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	h := cmd.HelmOutput{}
	err = yaml.Unmarshal(outBytes, &h)
	assert.NoErrorf(t, err, "Error parsing Yaml")
	assert.NotEmpty(t, h.Resources)
}

func TestGetAppsResourcesStatusRawFormat(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	name2, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace:  namespace,
		ShowStatus: true,
	}

	testDataPath := filepath.Join("test_data", "get_apps", "get_app_status.json")
	_, err = os.Stat(testDataPath)
	assert.NoError(t, err)
	expectedJSON, _ := ioutil.ReadFile(testDataPath)

	pegomock.When(testOptions.MockHelmer.StatusReleaseWithOutput(pegomock.EqString(namespace),
		pegomock.AnyString(), pegomock.EqString("json"))).
		ThenReturn(string(expectedJSON), nil)
	getAppOptions.Args = []string{name1, name2}
	console := tests.NewTerminal(t)
	r, fakeStdout, _ := os.Pipe()
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	err = getAppOptions.Run()
	assert.NoError(t, err)

	// check output
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	assert.NotEmpty(t, string(outBytes))
}

func TestGetAppNotFound(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	_, _, _, err := testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	_, _, _, err = testOptions.AddApp(nil, "")
	assert.NoError(t, err)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace: namespace,
	}
	r, fakeStdout, _ := os.Pipe()
	console := tests.NewTerminal(t)
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = fakeStdout
	getAppOptions.CommonOptions.Err = console.Err
	getAppOptions.Args = []string{"cheese"}
	err = getAppOptions.Run()
	fakeStdout.Close()
	r.Close()
	assert.EqualError(t, err, "No Apps found")
}
