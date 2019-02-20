package cmd_test

import (
	"github.com/Netflix/go-expect"
	"github.com/acarl005/stripansi"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd/cmd_test_helpers"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/petergtz/pegomock"
	"github.com/satori/go.uuid"
	"io/ioutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
)

func TestGetAppsGitops(t *testing.T) {
	t.SkipNow()
	// TODO Get gitops working
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(true, t)
	namespace := "jx-testing"

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1 := uuid.NewV4().String()
	name2 := uuid.NewV4().String()
	addApp(t, name1, namespace, testOptions)
	addApp(t, name2, namespace, testOptions)
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
	err := getAppOptions.Run()
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

func TestGetApps(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	namespace := "jx-testing"

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1 := uuid.NewV4().String()
	name2 := uuid.NewV4().String()
	addApp(t, name1, namespace, testOptions)
	addApp(t, name2, namespace, testOptions)
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
	err := getAppOptions.Run()
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
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	namespace := "jx-testing"

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1 := uuid.NewV4().String()
	name2 := uuid.NewV4().String()
	addApp(t, name1, namespace, testOptions)
	addApp(t, name2, namespace, testOptions)
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
	err := getAppOptions.Run()
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
	pegomock.RegisterMockTestingT(t)
	testOptions := cmd_test_helpers.CreateAppTestOptions(false, t)
	namespace := "jx-testing"

	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name1 := uuid.NewV4().String()
	name2 := uuid.NewV4().String()
	addApp(t, name1, namespace, testOptions)
	addApp(t, name2, namespace, testOptions)
	getAppOptions := &cmd.GetAppsOptions{
		GetOptions: cmd.GetOptions{
			CommonOptions: testOptions.CommonOptions,
		},
		Namespace: namespace,
	}
	console := tests.NewTerminal(t)
	getAppOptions.CommonOptions.In = console.In
	getAppOptions.CommonOptions.Out = console.Out
	getAppOptions.CommonOptions.Err = console.Err
	getAppOptions.Args = []string{"cheese"}
	err := getAppOptions.Run()
	assert.EqualError(t, err, "No Apps found in "+namespace+"\n")
}

func addApp(t *testing.T, name string, namespace string, testOptions *cmd_test_helpers.AppTestOptions) {
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
		GitOps:               false,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
		Namespace:            namespace,
	}
	o.Args = []string{name}
	err := o.Run()
	assert.NoError(t, err)
}
