// +build unit

package importcmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"

	jenkinsio "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io"
	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/v2/pkg/auth"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/jenkins-x/jx/v2/pkg/prow"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const testUsername = "derek_zoolander"

func TestCreateProwOwnersFileExistsDoNothing(t *testing.T) {
	t.Parallel()
	path, err := ioutil.TempDir("", "prow")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(path)
	ownerFilePath := filepath.Join(path, "OWNERS")
	_, err = os.Create(ownerFilePath)
	if err != nil {
		panic(err)
	}

	cmd := ImportOptions{
		Dir: path,
	}

	err = cmd.CreateProwOwnersFile()
	assert.NoError(t, err, "There should be no error")
}

func TestCreateProwOwnersFileCreateWhenDoesNotExist(t *testing.T) {
	t.Parallel()
	path, err := ioutil.TempDir("", "prow")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(path)

	cmd := ImportOptions{
		Dir: path,
		GitUserAuth: &auth.UserAuth{
			Username: testUsername,
		},
	}

	err = cmd.CreateProwOwnersFile()
	assert.NoError(t, err, "There should be no error")

	wantFile := filepath.Join(path, "OWNERS")
	exists, err := util.FileExists(wantFile)
	assert.NoError(t, err, "It should find the OWNERS file without error")
	assert.True(t, exists, "It should create an OWNERS file")

	wantOwners := prow.Owners{
		Approvers: []string{testUsername},
		Reviewers: []string{testUsername},
	}
	data, err := ioutil.ReadFile(wantFile)
	assert.NoError(t, err, "It should read the OWNERS file without error")
	owners := prow.Owners{}
	err = yaml.Unmarshal(data, &owners)
	assert.NoError(t, err, "It should unmarshal the OWNERS file without error")
	assert.Equal(t, wantOwners, owners)
}

func TestCreateProwOwnersFileCreateWhenDoesNotExistAndNoGitUserSet(t *testing.T) {
	t.Parallel()
	path, err := ioutil.TempDir("", "prow")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(path)

	cmd := ImportOptions{
		Dir: path,
	}

	err = cmd.CreateProwOwnersFile()
	assert.Error(t, err, "There should an error")
}

func TestCreateProwOwnersAliasesFileExistsDoNothing(t *testing.T) {
	t.Parallel()
	path, err := ioutil.TempDir("", "prow")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(path)
	ownerFilePath := filepath.Join(path, "OWNERS_ALIASES")
	_, err = os.Create(ownerFilePath)
	if err != nil {
		panic(err)
	}

	cmd := ImportOptions{
		Dir: path,
	}

	err = cmd.CreateProwOwnersAliasesFile()
	assert.NoError(t, err, "There should be no error")
}

func TestCreateProwOwnersAliasesFileCreateWhenDoesNotExist(t *testing.T) {
	t.Parallel()
	path, err := ioutil.TempDir("", "prow")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(path)
	cmd := ImportOptions{
		Dir: path,
		GitUserAuth: &auth.UserAuth{
			Username: testUsername,
		},
	}

	err = cmd.CreateProwOwnersAliasesFile()
	assert.NoError(t, err, "There should be no error")

	wantFile := filepath.Join(path, "OWNERS_ALIASES")
	exists, err := util.FileExists(wantFile)
	assert.NoError(t, err, "It should find the OWNERS_ALIASES file without error")
	assert.True(t, exists, "It should create an OWNERS_ALIASES file")

	wantOwnersAliases := prow.OwnersAliases{
		Aliases:       []string{testUsername},
		BestApprovers: []string{testUsername},
		BestReviewers: []string{testUsername},
	}
	data, err := ioutil.ReadFile(wantFile)
	assert.NoError(t, err, "It should read the OWNERS_ALIASES file without error")
	ownersAliases := prow.OwnersAliases{}
	err = yaml.Unmarshal(data, &ownersAliases)
	assert.NoError(t, err, "It should unmarshal the OWNERS_ALIASES file without error")
	assert.Equal(t, wantOwnersAliases, ownersAliases)
}

func TestCreateProwOwnersAliasesFileCreateWhenDoesNotExistAndNoGitUserSet(t *testing.T) {
	t.Parallel()
	path, err := ioutil.TempDir("", "prow")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(path)

	cmd := ImportOptions{
		Dir: path,
	}

	err = cmd.CreateProwOwnersAliasesFile()
	assert.Error(t, err, "There should an error")
}

func TestImportOptions_GetOrganisation(t *testing.T) {
	tests := []struct {
		name    string
		options ImportOptions
		want    string
	}{
		{
			name: "Get org from github URL (ignore user-specified org)",
			options: ImportOptions{
				RepoURL:      "https://github.com/orga/myrepo",
				Organisation: "orgb",
			},
			want: "orga",
		},
		{
			name: "Get org from github URL (no user-specified org)",
			options: ImportOptions{
				RepoURL: "https://github.com/orga/myrepo",
			},
			want: "orga",
		},
		{
			name: "Get org from user flag",
			options: ImportOptions{
				RepoURL:      "https://myrepo.com/myrepo", // No org here
				Organisation: "orgb",
			},
			want: "orgb",
		},
		{
			name: "No org specified",
			options: ImportOptions{
				RepoURL: "https://myrepo.com/myrepo", // No org here
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.options.GetOrganisation(); got != tt.want {
				t.Errorf("ImportOptions.GetOrganisation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetPreviewNamespace(t *testing.T) {
	t.Parallel()
	factory := fake.NewFakeFactory()
	commonOpts := opts.NewCommonOptionsWithFactory(factory)
	options := &commonOpts
	testhelpers.ConfigureTestOptions(options, gits.NewGitFake(), options.Helm())

	wd, err := os.Getwd()
	assert.NoError(t, err)
	testFolderPath := filepath.Join(wd, "test_data", "application_with_preview")
	tmpDirPath, err := ioutil.TempDir("", "preview_check")
	assert.NoError(t, err)
	defer func() {
		err = os.RemoveAll(tmpDirPath)
	}()
	err = util.CopyDir(testFolderPath, tmpDirPath, true)
	assert.NoError(t, err)

	o := &ImportOptions{
		CommonOptions:    &commonOpts,
		PreviewNamespace: "preview-ns",
		Dir:              tmpDirPath,
	}

	err = o.setPreviewNamespace()
	assert.NoError(t, err)

	previewsValuesFilePath := filepath.Join(o.Dir, "charts", "preview", opts.ValuesFile)
	exists, err := util.FileExists(previewsValuesFilePath)
	assert.NoError(t, err)
	assert.True(t, exists)
	values, err := ioutil.ReadFile(previewsValuesFilePath)
	assert.NoError(t, err)
	var valuesMap map[string]interface{}
	err = yaml.Unmarshal(values, &valuesMap)

	assert.Equal(t, "preview-ns", valuesMap["preview"].(map[string]interface{})["namespace"])
}

func TestSetPreviewNamespaceWithPermanentNs(t *testing.T) {
	t.Parallel()
	factory := fake.NewFakeFactory()
	commonOpts := opts.NewCommonOptionsWithFactory(factory)
	options := &commonOpts
	testhelpers.ConfigureTestOptions(options, gits.NewGitFake(), options.Helm())
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testFolderPath := filepath.Join(wd, "test_data", "application_with_preview")
	tmpDirPath, err := ioutil.TempDir("", "preview_check")
	assert.NoError(t, err)
	defer func() {
		err = os.RemoveAll(tmpDirPath)
	}()
	err = util.CopyDir(testFolderPath, tmpDirPath, true)
	assert.NoError(t, err)

	o := &ImportOptions{
		CommonOptions:    &commonOpts,
		PreviewNamespace: "jx",
		Dir:              tmpDirPath,
	}

	err = o.setPreviewNamespace()
	assert.NoError(t, err)

	previewsValuesFilePath := filepath.Join(o.Dir, "charts", "preview", opts.ValuesFile)
	exists, err := util.FileExists(previewsValuesFilePath)
	assert.NoError(t, err)
	assert.True(t, exists)
	values, err := ioutil.ReadFile(previewsValuesFilePath)
	assert.NoError(t, err)
	var valuesMap map[string]interface{}
	err = yaml.Unmarshal(values, &valuesMap)
	_, keyExists := valuesMap["preview"].(map[string]interface{})["namespace"]
	assert.False(t, keyExists, "the key \"namespace\" shouldn't exist in the values map")
}

func TestWriteSourceRepoToYaml(t *testing.T) {
	t.Parallel()
	path, err := ioutil.TempDir("", "test-source-repo-to-yaml")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(path)

	outDir := filepath.Join(path, "repositories", "templates")

	sr := &v1.SourceRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins-x-jx",
			Namespace: "jx",
			Labels:    map[string]string{"owner": "jenkins-x", "repository": "jx"},
		},

		Spec: v1.SourceRepositorySpec{
			Org:          "jenkins-x",
			Repo:         "jx",
			Provider:     gits.GitHubURL,
			ProviderName: gits.KindGitHub,
		},
	}

	err = writeSourceRepoToYaml(path, sr)
	assert.NoError(t, err)

	srFileName := filepath.Join(outDir, "jenkins-x-jx-sr.yaml")
	exists, err := util.FileExists(srFileName)
	assert.NoError(t, err)
	assert.True(t, exists, "serialized SR %s does not exist", srFileName)

	data, err := ioutil.ReadFile(srFileName)
	assert.NoError(t, err)

	newSr := &v1.SourceRepository{}

	err = yaml.Unmarshal(data, newSr)
	assert.NoError(t, err)

	assert.Equal(t, jenkinsio.GroupAndVersion, newSr.APIVersion)
	assert.Equal(t, "SourceRepository", newSr.Kind)
	assert.Equal(t, "jenkins-x-jx", newSr.Name)
	assert.Equal(t, "", newSr.Namespace)
}
