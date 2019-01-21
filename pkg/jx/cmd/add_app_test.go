package cmd_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"

	google_protobuf "github.com/golang/protobuf/ptypes/any"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
)

func TestAddAppForGitOps(t *testing.T) {
	t.Parallel()
	testOptions, err := cmd.CreateAppTestOptions(true)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
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
	}
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

func TestAddAppWithValuesFileForGitOps(t *testing.T) {
	t.Parallel()
	testOptions, err := cmd.CreateAppTestOptions(true)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

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
	testOptions, err := cmd.CreateAppTestOptions(true)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

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
	testOptions, err := cmd.CreateAppTestOptions(true)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
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
	err = o.Run()
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
	testOptions, err := cmd.CreateAppTestOptions(true)
	defer func() {
		err := testOptions.Cleanup()
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

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
