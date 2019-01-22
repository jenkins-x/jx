package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/satori/go.uuid"
	"k8s.io/apimachinery/pkg/runtime"
)

// AppTestOptions contains all useful data from the test environment initialized by `prepareInitialPromotionEnv`
type AppTestOptions struct {
	ConfigureGitFn  ConfigureGitFolderFn
	CommonOptions   *CommonOptions
	FakeGitProvider *gits.FakeProvider
	DevRepo         *gits.FakeRepository
	DevEnvRepo      *gits.FakeRepository
	OrgName         string
	DevEnvRepoInfo  *gits.GitRepository
	DevEnv          *v1.Environment
	MockHelmer      *helm_test.MockHelmer
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
	err := CleanupTestEnvironmentDir(o.CommonOptions)
	if err != nil {
		return err
	}
	return nil
}

// CreateAppTestOptions configures the mock environment for running apps related tests
func CreateAppTestOptions(gitOps bool) (*AppTestOptions, error) {
	o := AppTestOptions{
		CommonOptions: &CommonOptions{},
	}
	testOrgName := uuid.NewV4().String()
	testRepoName := uuid.NewV4().String()
	devEnvRepoName := fmt.Sprintf("environment-%s-%s-dev", testOrgName, testRepoName)
	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	devEnvRepo := gits.NewFakeRepository(testOrgName, devEnvRepoName)

	fakeGitProvider := gits.NewFakeProvider(fakeRepo, devEnvRepo)

	devEnv := kube.NewPermanentEnvironmentWithGit("dev", fmt.Sprintf("https://fake.git/%s/%s.git", testOrgName,
		devEnvRepoName))
	if gitOps {
		devEnv.Spec.Source.URL = devEnvRepo.GitRepo.CloneURL
		devEnv.Spec.Source.Ref = "master"
	}
	o.MockHelmer = helm_test.NewMockHelmer()

	ConfigureTestOptionsWithResources(o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			devEnv,
		},
		gits.NewGitLocal(),
		fakeGitProvider,
		o.MockHelmer,
	)

	err := CreateTestEnvironmentDir(o.CommonOptions)
	if err != nil {
		return nil, err
	}
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
	return &o, nil

}
