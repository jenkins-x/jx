package operations

import (
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/dependencymatrix"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	"github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreatePullRequest(t *testing.T) {

	_, _, _, commonOpts, _ := getFakeClientsAndNs(t)

	testOrgName := "testowner"
	testRepoName := "testrepo"

	commonOpts.SetGit(gits.NewGitFake())

	o := PullRequestOperation{
		CommonOptions: &commonOpts,
	}

	gitter := gits_test.NewMockGitter()

	fakeRepo, _ := gits.NewFakeRepository(testOrgName, testRepoName, nil, nil)
	fakeGitProvider := gits.NewFakeProvider(fakeRepo)
	fakeGitProvider.User.Username = testOrgName

	testhelpers.ConfigureTestOptionsWithResources(o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{},
		gitter,
		fakeGitProvider,
		nil,
		resources_test.NewMockInstaller(),
	)

	err := testhelpers.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)

	o.GitURLs = []string{"testowner/testrepo"}
	o.SrcGitURL = "testowner/testrepo"
	o.Version = "3.0.0"

	pegomock.When(gitter.HasChanges(pegomock.AnyString())).ThenReturn(true, nil)

	var results *gits.PullRequestInfo
	logOutput := log.CaptureOutput(func() {
		results, err = o.CreatePullRequest("test", func(dir string, gitInfo *gits.GitRepository) (strings []string, e error) {
			return []string{"1.0.0", "v1.0.1", "2.0.0"}, nil
		})
		assert.NoError(t, err)
	})

	assert.Contains(t, logOutput, "Added label updatebot to Pull Request https://fake.git/testowner/testrepo/pulls/1",
		"Updatebot label should be added to the PR")
	assert.NotNil(t, results, "we must have results coming out of the PR creation")
	assert.Equal(t, "chore(deps): bump testowner/testrepo from 1.0.0, 2.0.0 and v1.0.1 to 3.0.0",
		results.PullRequestArguments.Title, "The PR title should contain the old and new versions")
}

func TestCreatePullRequestWithMatrixUpdatePaths(t *testing.T) {

	_, _, _, commonOpts, _ := getFakeClientsAndNs(t)

	testOrgName := "testowner"
	testRepoName := "testrepo"

	commonOpts.SetGit(gits.NewGitFake())
	o := PullRequestOperation{
		CommonOptions: &commonOpts,
	}

	viaRepo := "wiley"
	toVersion := "3.0.0"
	fromVersion := "1.0.0"
	toTag := fmt.Sprintf("v%s", toVersion)
	fromTag := fmt.Sprintf("v%s", fromVersion)
	host := "fake.git"
	updates := dependencymatrix.DependencyUpdates{
		Updates: []v1.DependencyUpdate{
			{
				DependencyUpdateDetails: v1.DependencyUpdateDetails{
					Host:               host,
					Owner:              testOrgName,
					Repo:               testRepoName,
					URL:                fmt.Sprintf("https://%s/%s/%s.git", host, testOrgName, testRepoName),
					ToReleaseHTMLURL:   fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, testRepoName, toTag),
					ToVersion:          toVersion,
					ToReleaseName:      toVersion,
					FromReleaseHTMLURL: fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, testRepoName, fromTag),
					FromReleaseName:    fromVersion,
					FromVersion:        fromVersion,
				},
				Paths: []v1.DependencyUpdatePath{
					{
						{
							Host:               host,
							Owner:              testOrgName,
							Repo:               viaRepo,
							URL:                fmt.Sprintf("https://%s/%s/%s.git", host, testOrgName, viaRepo),
							ToReleaseHTMLURL:   fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, viaRepo, toTag),
							ToVersion:          toVersion,
							ToReleaseName:      toVersion,
							FromReleaseHTMLURL: fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, viaRepo, fromTag),
							FromReleaseName:    fromVersion,
							FromVersion:        fromVersion,
						},
					},
				},
			},
		},
	}

	updateBytes, err := yaml.Marshal(updates)
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(200)
		fmt.Fprintf(w, string(updateBytes))
	}))

	gitter := gits_test.NewMockGitter()

	fakeRepo, _ := gits.NewFakeRepository(testOrgName, testRepoName, nil, nil)
	fakeRepo.Releases = map[string]*gits.GitRelease{
		"release-1": {
			Name:    "Release 1",
			TagName: "1.0.0",
			HTMLURL: "fakeUrlv1",
		},
		"release-3": {
			Name:    "Release 3",
			TagName: "3.0.0",
			HTMLURL: "fakeUrlv3",
			Assets: &[]gits.GitReleaseAsset{
				{
					ID:                 1,
					Name:               dependencymatrix.DependencyUpdatesAssetName,
					BrowserDownloadURL: server.URL,
					ContentType:        "application/json",
				},
			},
		},
	}
	fakeGitProvider := gits.NewFakeProvider(fakeRepo)
	fakeGitProvider.User.Username = testOrgName

	testhelpers.ConfigureTestOptionsWithResources(o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{},
		gitter,
		fakeGitProvider,
		nil,
		resources_test.NewMockInstaller(),
	)

	err = testhelpers.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)

	o.GitURLs = []string{"testowner/testrepo"}
	o.SrcGitURL = "testowner/testrepo"
	o.Version = "3.0.0"

	pegomock.When(gitter.HasChanges(pegomock.AnyString())).ThenReturn(true, nil)

	var results *gits.PullRequestInfo
	logOutput := log.CaptureOutput(func() {
		results, err = o.CreatePullRequest("test", func(dir string, gitInfo *gits.GitRepository) (strings []string, e error) {
			return []string{"1.0.0", "v1.0.1", "2.0.0"}, nil
		})
		assert.NoError(t, err)
	})

	assert.Contains(t, stripansi.Strip(logOutput), "Added label updatebot to Pull Request https://fake.git/testowner/testrepo/pulls/1",
		"Updatebot label should be added to the PR")
	assert.NotNil(t, results, "we must have results coming out of the PR creation")
	assert.Equal(t, "chore(deps): bump testowner/testrepo from 1.0.0, 2.0.0 and v1.0.1 to 3.0.0",
		results.PullRequestArguments.Title, "The PR title should contain the old and new versions")
}

func TestCreateDependencyUpdatePRDetails(t *testing.T) {
	_, _, _, commonOpts, _ := getFakeClientsAndNs(t)

	commonOpts.SetGit(gits.NewGitFake())
	o := PullRequestOperation{
		CommonOptions: &commonOpts,
	}

	gitter := gits_test.NewMockGitter()

	testOrgName := "testowner"
	testRepoName := "testrepo"
	gitRepo := fmt.Sprintf("%s/%s", testOrgName, testRepoName)
	fakeRepo, _ := gits.NewFakeRepository(testOrgName, testRepoName, nil, nil)
	fakeRepo.Releases = map[string]*gits.GitRelease{
		"release-1": {
			Name:    "Release 1",
			TagName: "1.0.0",
			HTMLURL: "fakeUrlv1",
		},
		"release-2": {
			Name:    "Release 2",
			TagName: "2.0.0",
			HTMLURL: "fakeUrlv2",
			Assets: &[]gits.GitReleaseAsset{
				{
					ID:                 1,
					Name:               "Asset1",
					BrowserDownloadURL: "fakeURL",
					ContentType:        "application/json",
				},
			},
		},
	}
	fakeGitProvider := gits.NewFakeProvider(fakeRepo)
	fakeGitProvider.User.Username = testOrgName

	testhelpers.ConfigureTestOptionsWithResources(o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{},
		gitter,
		fakeGitProvider,
		nil,
		resources_test.NewMockInstaller(),
	)

	err := testhelpers.CreateTestEnvironmentDir(o.CommonOptions)
	assert.NoError(t, err)

	componentName := "testComponent"
	fromVersion := "1.0.0"
	toVersion := "2.0.0"
	kind := "fakekind"
	_, details, _, assets, err := o.CreateDependencyUpdatePRDetails(kind, gitRepo, &gits.GitRepository{}, fromVersion, toVersion, componentName)

	assert.Contains(t, details.BranchName, fmt.Sprintf("bump-%s-version", kind))
	assert.Contains(t, details.Message, fmt.Sprintf("Update [%s](%s):%s from [%s](fakeUrlv1) to [%s](fakeUrlv2)", gitRepo, gitRepo, componentName, fromVersion, toVersion))
	assert.Len(t, assets, 1)
}

func TestAddDependencyMatrixUpdatePaths(t *testing.T) {
	testOrgName := "testowner"
	testRepoName := "testrepo"
	viaRepo := "wiley"
	toVersion := "3.0.0"
	fromVersion := "1.0.0"
	toTag := fmt.Sprintf("v%s", toVersion)
	fromTag := fmt.Sprintf("v%s", fromVersion)
	host := "fake.git"
	updates := dependencymatrix.DependencyUpdates{
		Updates: []v1.DependencyUpdate{
			{
				DependencyUpdateDetails: v1.DependencyUpdateDetails{
					Host:               host,
					Owner:              testOrgName,
					Repo:               testRepoName,
					URL:                fmt.Sprintf("https://%s/%s/%s.git", host, testOrgName, testRepoName),
					ToReleaseHTMLURL:   fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, testRepoName, toTag),
					ToVersion:          toVersion,
					ToReleaseName:      toVersion,
					FromReleaseHTMLURL: fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, testRepoName, fromTag),
					FromReleaseName:    fromVersion,
					FromVersion:        fromVersion,
				},
				Paths: []v1.DependencyUpdatePath{
					{
						{
							Host:               host,
							Owner:              testOrgName,
							Repo:               viaRepo,
							URL:                fmt.Sprintf("https://%s/%s/%s.git", host, testOrgName, viaRepo),
							ToReleaseHTMLURL:   fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, viaRepo, toTag),
							ToVersion:          toVersion,
							ToReleaseName:      toVersion,
							FromReleaseHTMLURL: fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, viaRepo, fromTag),
							FromReleaseName:    fromVersion,
							FromVersion:        fromVersion,
						},
					},
				},
			},
		},
	}

	updateBytes, err := yaml.Marshal(updates)
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(200)
		fmt.Fprintf(w, string(updateBytes))
	}))

	asset := &gits.GitReleaseAsset{
		ID:                 1,
		BrowserDownloadURL: server.URL,
		ContentType:        "application/yaml",
		Name:               dependencymatrix.DependencyUpdatesAssetName,
	}

	update := &v1.DependencyUpdate{
		DependencyUpdateDetails: v1.DependencyUpdateDetails{
			Host:               host,
			Owner:              testOrgName,
			Repo:               testRepoName,
			URL:                fmt.Sprintf("https://%s/%s/%s.git", host, testOrgName, testRepoName),
			ToReleaseHTMLURL:   fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, testRepoName, toTag),
			ToVersion:          "2,0,0",
			ToReleaseName:      "2,0,0",
			FromReleaseHTMLURL: fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, testRepoName, fromTag),
			FromReleaseName:    fromVersion,
			FromVersion:        fromVersion,
		},
		Paths: []v1.DependencyUpdatePath{
			{
				{
					Host:               host,
					Owner:              testOrgName,
					Repo:               viaRepo,
					URL:                fmt.Sprintf("https://%s/%s/%s.git", host, testOrgName, viaRepo),
					ToReleaseHTMLURL:   fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, viaRepo, toTag),
					ToVersion:          "2,0,0",
					ToReleaseName:      "2,0,0",
					FromReleaseHTMLURL: fmt.Sprintf("https://%s/%s/%s/releases/%s", host, testOrgName, viaRepo, fromTag),
					FromReleaseName:    fromVersion,
					FromVersion:        fromVersion,
				},
			},
		},
	}

	dependencyUpdate, err := addDependencyMatrixUpdatePaths(asset, update)
	assert.NoError(t, err)
	assert.Len(t, dependencyUpdate[0].Paths[0], 2)
}

// Helper method, not supposed to be a test by itself
func getFakeClientsAndNs(t *testing.T) (versioned.Interface, tektonclient.Interface, kubernetes.Interface, opts.CommonOptions, string) {
	commonOpts := opts.NewCommonOptionsWithFactory(fake.NewFakeFactory())
	options := &commonOpts
	testhelpers.ConfigureTestOptions(options, options.Git(), options.Helm())

	jxClient, ns, err := options.JXClientAndDevNamespace()
	assert.NoError(t, err, "There shouldn't be any error getting the fake JXClient and DevEnv")

	tektonClient, _, err := options.TektonClient()
	assert.NoError(t, err, "There shouldn't be any error getting the fake Tekton Client")

	kubeClient, err := options.KubeClient()
	assert.NoError(t, err, "There shouldn't be any error getting the fake Kube Client")

	return jxClient, tektonClient, kubeClient, commonOpts, ns
}
