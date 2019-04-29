package cmd_test

import (
	"errors"
	"github.com/jenkins-x/jx/pkg/helm"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/acarl005/stripansi"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/cmd_test_helpers"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/petergtz/pegomock"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetAppsGitops(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	name1UUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name1 := name1UUID.String()
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, name1, t)

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	_, _, _, err = testOptions.DirectlyAddAppToGitOps(name1, nil, "jx-app")
	assert.NoError(t, err)

	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
	}
	getAppOptions.ConfigureGitFn = testOptions.ConfigureGitFn

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
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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

	pegomock.When(getAppOptions.Helm().ListReleases(pegomock.EqString(namespace))).
		ThenReturn(map[string]helm.ReleaseSummary{
			name1: {Status: "DEPLOYED"},
		}, nil, nil)

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
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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

	pegomock.When(getAppOptions.Helm().ListReleases(pegomock.EqString(namespace))).
		ThenReturn(map[string]helm.ReleaseSummary{
			name1: {Status: "DEPLOYED"},
		}, nil, nil)

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
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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

	pegomock.When(getAppOptions.Helm().ListReleases(pegomock.EqString(namespace))).
		ThenReturn(map[string]helm.ReleaseSummary{
			name1: {Status: "DEPLOYED"},
		}, nil, nil)

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

func TestGetAppNotFound(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, "", t)

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
