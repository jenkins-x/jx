package cmd_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/helm/pkg/chartutil"

	vault_test "github.com/jenkins-x/jx/pkg/vault/mocks"

	expect "github.com/Netflix/go-expect"
	"github.com/jenkins-x/jx/pkg/tests"

	"github.com/jenkins-x/jx/pkg/gits"
	cmd_test "github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/petergtz/pegomock"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ghodss/yaml"

	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/jenkins-x/jx/pkg/helm"

	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"

	google_protobuf "github.com/golang/protobuf/ptypes/any"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	installer_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
)

func TestAddAppForGitOps(t *testing.T) {
	t.Parallel()
	testOptions := CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name := uuid.NewV4().String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
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
	err := o.Run()
	assert.NoError(t, err)
	pr, err := testOptions.FakeGitProvider.GetPullRequest(testOptions.OrgName, testOptions.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
	// Validate the PR has the right title, message
	assert.Equal(t, fmt.Sprintf("Add %s %s", name, version), pr.Title)
	assert.Equal(t, fmt.Sprintf("Add app %s %s", name, version), pr.Body)
	// Validate the branch name
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
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

func TestAddAppWithSecrets(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := CreateAppTestOptions(false, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	// Needs console input to create secrets
	console := tests.NewTerminal(t)
	testOptions.CommonOptions.In = console.In
	testOptions.CommonOptions.Out = console.Out
	testOptions.CommonOptions.Err = console.Err

	name := uuid.NewV4().String()
	version := "0.0.1"
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
		},
		Version:              version,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}

	helm_test.StubFetchChart(name, "", cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		Files: []*google_protobuf.Any{
			&google_protobuf.Any{
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
		pegomock.EqString(cmd.DEFAULT_CHARTMUSEUM_URL),
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
			assert.Contains(t, secretsFileName, "secrets.yaml")
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

	err := o.Run()
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
			pegomock.EqString(cmd.DEFAULT_CHARTMUSEUM_URL),
			pegomock.AnyString(),
			pegomock.AnyString())
}

func TestAddAppForGitOpsWithSecrets(t *testing.T) {
	tests.SkipForWindows(t, "go-expect does not work on windows")
	pegomock.RegisterMockTestingT(t)
	testOptions := CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	// Needs console input to create secrets
	console := tests.NewTerminal(t)
	testOptions.CommonOptions.In = console.In
	testOptions.CommonOptions.Out = console.Out
	testOptions.CommonOptions.Err = console.Err

	name := uuid.NewV4().String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
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

	helm_test.StubFetchChart(name, "", cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		Files: []*google_protobuf.Any{
			&google_protobuf.Any{
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
	err := o.Run()
	assert.NoError(t, err)
	err = console.Close()
	<-donec
	assert.NoError(t, err)
	t.Logf(expect.StripTrailingEmptyLines(console.CurrentState()))

	// Validate that the secret reference is generated
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
	valuesFromPrPath := filepath.Join(devEnvDir, name, helm.ValuesFileName)
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
	testOptions := CreateAppTestOptions(false, t)
	// Can't run in parallel
	pegomock.RegisterMockTestingT(t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name := uuid.NewV4().String()
	version := "0.0.1"
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
		},
		Version:              version,
		Repo:                 cmd.DEFAULT_CHARTMUSEUM_URL,
		GitOps:               false,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	err := o.Run()
	assert.NoError(t, err)

	_, _, _, fetchDir, _, _, _ := testOptions.MockHelmer.VerifyWasCalledOnce().FetchChart(
		pegomock.EqString(name),
		pegomock.EqString(version),
		pegomock.AnyBool(),
		pegomock.AnyString(),
		pegomock.EqString(cmd.DEFAULT_CHARTMUSEUM_URL),
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
			pegomock.EqString(cmd.DEFAULT_CHARTMUSEUM_URL),
			pegomock.AnyString(),
			pegomock.AnyString())
}

func TestAddLatestApp(t *testing.T) {

	testOptions := CreateAppTestOptions(false, t)
	// Can't run in parallel
	pegomock.RegisterMockTestingT(t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name := uuid.NewV4().String()
	version := "0.1.1"
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
		},
		Repo:                 cmd.DEFAULT_CHARTMUSEUM_URL,
		GitOps:               false,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	helm_test.StubFetchChart(name, "", cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
	}, testOptions.MockHelmer)
	err := o.Run()
	assert.NoError(t, err)

	_, _, _, fetchDir, _, _, _ := testOptions.MockHelmer.VerifyWasCalledOnce().FetchChart(
		pegomock.EqString(name),
		pegomock.AnyString(),
		pegomock.AnyBool(),
		pegomock.AnyString(),
		pegomock.EqString(cmd.DEFAULT_CHARTMUSEUM_URL),
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
			pegomock.EqString(cmd.DEFAULT_CHARTMUSEUM_URL),
			pegomock.AnyString(),
			pegomock.AnyString())
}

func TestAddAppWithValuesFileForGitOps(t *testing.T) {
	t.Parallel()
	testOptions := CreateAppTestOptions(true, t)
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

	name := uuid.NewV4().String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
		},
		Version:              version,
		Alias:                alias,
		Repo:                 "http://chartmuseum.jenkins-x.io",
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
		ValueFiles:           []string{file.Name()},
	}
	o.Args = []string{name}
	err = o.Run()
	assert.NoError(t, err)
	// Validate that the values.yaml file is in the right place
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
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
	testOptions := CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name := uuid.NewV4().String()
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
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
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
	helm_test.StubFetchChart(name, "", cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:        name,
			Version:     version,
			Description: description,
		},
		Templates: []*chart.Template{
			&chart.Template{
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
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
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
`, name, version, description, cmd.DEFAULT_CHARTMUSEUM_URL, gitRepository, releaseNotes), readmeFromPr)
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
	testOptions := CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name := uuid.NewV4().String()
	version := "0.0.1"
	alias := fmt.Sprintf("%s-alias", name)
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
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
	helm_test.StubFetchChart(name, "", cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		Files: []*google_protobuf.Any{
			&google_protobuf.Any{
				TypeUrl: readmeFileName,
				Value:   []byte(readme),
			},
		},
	}, testOptions.MockHelmer)
	err := o.Run()
	assert.NoError(t, err)
	// Validate that the README.md file is in the right place
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
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
`, name, version, cmd.DEFAULT_CHARTMUSEUM_URL, readme), readmeFromPr)
}

func TestAddLatestAppForGitOps(t *testing.T) {
	testOptions := CreateAppTestOptions(true, t)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()

	name := uuid.NewV4().String()
	version := "0.1.8"
	alias := fmt.Sprintf("%s-alias", name)
	o := &cmd.AddAppOptions{
		AddOptions: cmd.AddOptions{
			CommonOptions: *testOptions.CommonOptions,
		},
		Alias:                alias,
		Repo:                 cmd.DEFAULT_CHARTMUSEUM_URL,
		GitOps:               true,
		DevEnv:               testOptions.DevEnv,
		HelmUpdate:           true, // Flag default when run on CLI
		ConfigureGitCallback: testOptions.ConfigureGitFn,
	}
	o.Args = []string{name}
	o.Verbose = true

	helm_test.StubFetchChart(name, "", cmd.DEFAULT_CHARTMUSEUM_URL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
	}, testOptions.MockHelmer)

	err := o.Run()
	assert.NoError(t, err)
	pr, err := testOptions.FakeGitProvider.GetPullRequest(testOptions.OrgName, testOptions.DevEnvRepoInfo, 1)
	assert.NoError(t, err)
	// Validate the PR has the right title, message
	assert.Equal(t, fmt.Sprintf("Add %s %s", name, version), pr.Title)
	assert.Equal(t, fmt.Sprintf("Add app %s %s", name, version), pr.Body)
	// Validate the branch name
	envDir, err := o.CommonOptions.EnvironmentsDir()
	assert.NoError(t, err)
	devEnvDir := filepath.Join(envDir, testOptions.OrgName, testOptions.DevEnvRepoInfo.Name)
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

// Helpers for various app tests

// AppTestOptions contains all useful data from the test environment initialized by `prepareInitialPromotionEnv`
type AppTestOptions struct {
	ConfigureGitFn  cmd.ConfigureGitFolderFn
	CommonOptions   *cmd.CommonOptions
	FakeGitProvider *gits.FakeProvider
	DevRepo         *gits.FakeRepository
	DevEnvRepo      *gits.FakeRepository
	OrgName         string
	DevEnvRepoInfo  *gits.GitRepository
	DevEnv          *jenkinsv1.Environment
	MockHelmer      *helm_test.MockHelmer
	MockFactory     *cmd_test.MockFactory
	MockVaultClient *vault_test.MockClient
}

// AddApp modifies the environment git repo directly to add a dummy app
func (o *AppTestOptions) AddApp() (name string, alias string, version string, err error) {
	envDir, err := o.CommonOptions.EnvironmentsDir()
	if err != nil {
		return "", "", "", err
	}
	devEnvDir := filepath.Join(envDir, o.OrgName, o.DevEnvRepoInfo.Name)
	err = os.MkdirAll(devEnvDir, 0700)
	if err != nil {
		return "", "", "", err
	}
	fileName := filepath.Join(devEnvDir, helm.RequirementsFileName)
	requirements := helm.Requirements{}
	if _, err := os.Stat(fileName); err == nil {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return "", "", "", err
		}

		err = yaml.Unmarshal(data, &requirements)
		if err != nil {
			return "", "", "", err
		}
	}
	name = uuid.NewV4().String()
	alias = fmt.Sprintf("%s-alias", name)
	version = "0.0.1"
	requirements.Dependencies = append(requirements.Dependencies, &helm.Dependency{
		Name:       name,
		Alias:      alias,
		Version:    version,
		Repository: "http://fake.chartmuseum",
	})
	data, err := yaml.Marshal(requirements)
	if err != nil {
		return "", "", "", err
	}
	err = ioutil.WriteFile(fileName, data, 0755)
	if err != nil {
		return "", "", "", err
	}
	return name, alias, version, nil
}

// Cleanup must be run in a defer statement whenever CreateAppTestOptions is run
func (o *AppTestOptions) Cleanup() error {
	err := cmd.CleanupTestEnvironmentDir(o.CommonOptions)
	if err != nil {
		return err
	}
	return nil
}

// CreateAppTestOptions configures the mock environment for running apps related tests
func CreateAppTestOptions(gitOps bool, t *testing.T) *AppTestOptions {
	mockFactory := cmd_test.NewMockFactory()
	o := AppTestOptions{
		CommonOptions: &cmd.CommonOptions{
			Factory: mockFactory,
		},
	}
	testOrgName := uuid.NewV4().String()
	testRepoName := uuid.NewV4().String()
	devEnvRepoName := fmt.Sprintf("environment-%s-%s-dev", testOrgName, testRepoName)
	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	devEnvRepo := gits.NewFakeRepository(testOrgName, devEnvRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, devEnvRepo)

	var devEnv *jenkinsv1.Environment
	if gitOps {
		devEnv = kube.NewPermanentEnvironmentWithGit("dev", fmt.Sprintf("https://fake.git/%s/%s.git", testOrgName,
			devEnvRepoName))
		devEnv.Spec.Source.URL = devEnvRepo.GitRepo.CloneURL
		devEnv.Spec.Source.Ref = "master"
		o.MockVaultClient = vault_test.NewMockClient()
		pegomock.When(mockFactory.UseVault()).ThenReturn(pegomock.ReturnValue(true))
		pegomock.When(mockFactory.CreateSystemVaultClient(pegomock.AnyString())).ThenReturn(pegomock.ReturnValue(o.
			MockVaultClient), pegomock.ReturnValue(nil))
	} else {
		devEnv = kube.NewPermanentEnvironment("dev")
	}
	o.MockHelmer = helm_test.NewMockHelmer()
	installerMock := installer_test.NewMockInstaller()
	cmd.ConfigureTestOptionsWithResources(o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			devEnv,
		},
		gits.NewGitLocal(),
		fakeGitProvider,
		o.MockHelmer,
		installerMock,
	)

	err := cmd.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)
	o.ConfigureGitFn = func(dir string, gitInfo *gits.GitRepository, gitter gits.Gitter) error {
		err := gitter.Init(dir)
		if err != nil {
			return err
		}
		// Really we should have a dummy environment chart but for now let's just mock it out as needed
		err = os.MkdirAll(filepath.Join(dir, "templates"), 0700)
		if err != nil {
			return err
		}
		data, err := json.Marshal(devEnv)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, "templates", "dev-env.yaml"), data, 0755)
		if err != nil {
			return err
		}
		return gitter.AddCommit(dir, "Initial Commit")
	}
	o.FakeGitProvider = fakeGitProvider
	o.DevRepo = fakeRepo
	o.DevEnvRepo = devEnvRepo
	o.OrgName = testOrgName
	o.DevEnv = devEnv
	o.DevEnvRepoInfo = &gits.GitRepository{
		Name: devEnvRepoName,
	}
	return &o

}
