package cmd_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/acarl005/stripansi"
	"github.com/ghodss/yaml"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd/cmd_test_helpers"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/petergtz/pegomock"
	uuid "github.com/satori/go.uuid"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/helm/pkg/proto/hapi/chart"

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

	name1UUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name1 := name1UUID.String()
	addApp(t, name1, namespace, testOptions, true)
	namespace = "jx-testing"
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
	appResourceLocation := filepath.Join(devEnvDir, name1, "templates", name1+"-app.yaml")
	app := &v1.App{}
	appBytes, err := ioutil.ReadFile(appResourceLocation)
	err = yaml.Unmarshal(appBytes, app)
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
	namespace := "jx-testing"

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1UUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name1 := name1UUID.String()
	name2UUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name2 := name2UUID.String()
	addApp(t, name1, namespace, testOptions, false)
	addApp(t, name2, namespace, testOptions, false)
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

func TestGetApp(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	namespace := "jx-testing"

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1UUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name1 := name1UUID.String()
	name2UUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name2 := name2UUID.String()
	addApp(t, name1, namespace, testOptions, false)
	addApp(t, name2, namespace, testOptions, false)
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

func TestGetAppNotFound(t *testing.T) {
	tests.SkipForWindows(t, "NewTerminal() does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	namespace := "jx-testing"

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1UUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name1 := name1UUID.String()
	name2UUID, err := uuid.NewV4()
	assert.NoError(t, err)
	name2 := name2UUID.String()
	addApp(t, name1, namespace, testOptions, false)
	addApp(t, name2, namespace, testOptions, false)
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
	assert.NoError(t, err)
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, "No Apps found\n")
}

func addApp(t *testing.T, name string, namespace string, testOptions *cmd_test_helpers.AppTestOptions, gitOps bool) {
	version := "0.0.1"

	helm_test.StubFetchChart(name, version, kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:        name,
			Version:     version,
			Description: "My test chart description",
		},
	}, testOptions.MockHelmer)

	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Version:              version,
		Repo:                 kube.DefaultChartMuseumURL,
		GitOps:               gitOps,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
		Namespace:            namespace,
	}
	o.Args = []string{name}
	err := o.Run()
	assert.NoError(t, err)
}
