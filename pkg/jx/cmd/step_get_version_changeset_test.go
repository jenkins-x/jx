package cmd_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/jenkins-x/jx/pkg/version"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestStepGetVersionChangeSetOptionsBranch(t *testing.T) {
	t.Parallel()
	testDir, _ := ioutil.TempDir("", "test-version-changesetbranch")
	defer func() {
		err := os.RemoveAll(testDir)
		assert.NoError(t, err)
	}()
	repoOwnerUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	repoOwner := repoOwnerUUID.String()
	repoNameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	repoName := repoNameUUID.String()
	fakeRepo := gits.NewFakeRepository(repoOwner, repoName)
	fakeGitProvider := gits.NewFakeProvider(fakeRepo)
	testBranch := "test-app-version-bump"
	stableBranch := "master"
	r, fakeStdout, _ := os.Pipe()
	options := &cmd.StepGetVersionChangeSetOptions{
		StepOptions: cmd.StepOptions{
			CommonOptions: &opts.CommonOptions{},
		},
		VersionsDir:   testDir,
		TestingBranch: testBranch,
		StableBranch:  stableBranch,
	}
	options.CommonOptions.Out = fakeStdout
	cmd.ConfigureTestOptionsWithResources(options.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{},
		gits.NewGitLocal(),
		fakeGitProvider,
		helm.NewHelmCLI("helm", helm.V2, "", true),
		resources_test.NewMockInstaller(),
	)
	gitter := options.Git()
	gitter.Init(testDir)
	stableVersion := &version.StableVersion{
		Version: "1.0.0",
		GitURL:  "fake",
	}
	gitter.AddRemote(testDir, "origin", fakeRepo.GitRepo.CloneURL)
	version.SaveStableVersion(testDir, version.KindChart, "test-app", stableVersion)
	gitter.Add(testDir, ".")
	gitter.AddCommit(testDir, "Initial Commit")
	gitter.Push(testDir)
	gitter.CreateBranch(testDir, testBranch)
	gitter.Checkout(testDir, testBranch)
	stableVersion.Version = "1.0.1"
	version.SaveStableVersion(testDir, version.KindChart, "test-app", stableVersion)
	gitter.AddCommit(testDir, "Bump version")
	gitter.Push(testDir)
	gitter.Checkout(testDir, stableBranch)

	err = options.Run()
	assert.NoError(t, err)
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, `JX_CHANGED_VERSIONS="charts:test-app:1.0.1"`)
	assert.Contains(t, output, `JX_STABLE_VERSIONS="charts:test-app:1.0.0"`)

}

func TestStepGetVersionChangeSetOptionsPR(t *testing.T) {
	t.Parallel()
	testDir, _ := ioutil.TempDir("", "test-version-changesetpr")
	defer func() {
		err := os.RemoveAll(testDir)
		assert.NoError(t, err)
	}()
	repoOwnerUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	repoOwner := repoOwnerUUID.String()
	repoNameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	repoName := repoNameUUID.String()
	fakeRepo := gits.NewFakeRepository(repoOwner, repoName)
	fakeGitProvider := gits.NewFakeProvider(fakeRepo)
	testBranch := "test-app-version-bump"
	stableBranch := "master"
	r, fakeStdout, _ := os.Pipe()
	options := &cmd.StepGetVersionChangeSetOptions{
		StepOptions: cmd.StepOptions{
			CommonOptions: &opts.CommonOptions{},
		},
		VersionsDir:  testDir,
		PR:           "1",
		StableBranch: stableBranch,
	}
	options.CommonOptions.Out = fakeStdout
	cmd.ConfigureTestOptionsWithResources(options.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{},
		gits.NewGitLocal(),
		fakeGitProvider,
		helm.NewHelmCLI("helm", helm.V2, "", true),
		resources_test.NewMockInstaller(),
	)
	gitter := options.Git()
	gitter.Init(testDir)
	stableVersion := &version.StableVersion{
		Version: "1.0.0",
		GitURL:  "fake",
	}
	gitter.AddRemote(testDir, "origin", fakeRepo.GitRepo.CloneURL)
	version.SaveStableVersion(testDir, version.KindChart, "test-app", stableVersion)
	gitter.Add(testDir, ".")
	gitter.AddCommit(testDir, "Initial Commit")
	gitter.Push(testDir)
	gitter.CreateBranch(testDir, testBranch)
	gitter.Checkout(testDir, testBranch)
	stableVersion.Version = "1.0.1"
	version.SaveStableVersion(testDir, version.KindChart, "test-app", stableVersion)
	gitter.AddCommit(testDir, "Bump version")
	gitter.CreateBranch(testDir, "1")
	gitter.Checkout(testDir, stableBranch)

	err = options.Run()
	assert.NoError(t, err)
	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, `JX_CHANGED_VERSIONS="charts:test-app:1.0.1"`)
	assert.Contains(t, output, `JX_STABLE_VERSIONS="charts:test-app:1.0.0"`)

}
