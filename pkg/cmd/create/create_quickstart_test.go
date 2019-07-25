package create_test

import (
	"io"
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/cmd/create"
	"github.com/jenkins-x/jx/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/quickstarts"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func TestSetQuickstartPlaformSetsKubernetesIfEmptyAndIfBatchMode(t *testing.T) {
	// Setup
	commonOpts := &opts.CommonOptions{
		BatchMode: true,
	}
	opts := create.CreateQuickstartOptions{
		CreateProjectOptions: create.CreateProjectOptions{
			ImportOptions: importcmd.ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	// Execute
	opts.SetQuickstartPlatform()
	// Evaluate
	assert.Equal(t, "kubernetes", opts.Filter.Platform)
}

func TestSetQuickstartPlaformDoesNothingIfNotEmptyAndIfBatchMode(t *testing.T) {
	// Setup
	commonOpts := &opts.CommonOptions{
		BatchMode: true,
	}
	opts := create.CreateQuickstartOptions{
		CreateProjectOptions: create.CreateProjectOptions{
			ImportOptions: importcmd.ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	opts.Filter.Platform = "something"
	// Execute
	opts.SetQuickstartPlatform()
	// Evaluate
	assert.Equal(t, "something", opts.Filter.Platform)
}

func TestSetQuickstartPlaformInvokesPickNameWithDefaultWhenNotInBatchMode(t *testing.T) {
	// Setup
	pickNameWithDefaultOrig := util.PickNameWithDefault
	defer func() {
		util.PickNameWithDefault = pickNameWithDefaultOrig
	}()
	actualNames := []string{}
	actualMessage := ""
	actualDefaultValue := ""
	util.PickNameWithDefault = func(names []string, message string, defaultValue string, help string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) (string, error) {
		actualNames = names
		actualMessage = message
		actualDefaultValue = defaultValue
		return "something", nil
	}
	commonOpts := &opts.CommonOptions{}
	opts := create.CreateQuickstartOptions{
		CreateProjectOptions: create.CreateProjectOptions{
			ImportOptions: importcmd.ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	// Execute
	opts.SetQuickstartPlatform()
	// Evaluate
	assert.Equal(t, []string{"kubernetes", "serverless"}, actualNames)
	assert.Equal(t, "Pick the deployment platform:", actualMessage)
	assert.Equal(t, "kubernetes", actualDefaultValue)
	assert.Equal(t, "something", opts.Filter.Platform)
}

func TestSetQuickstartPlaformReturnsTheErrorFromPickNameWithDefault(t *testing.T) {
	// Setup
	pickNameWithDefaultOrig := util.PickNameWithDefault
	defer func() {
		util.PickNameWithDefault = pickNameWithDefaultOrig
	}()
	util.PickNameWithDefault = func(names []string, message string, defaultValue string, help string, in terminal.FileReader, out terminal.FileWriter, outErr io.Writer) (string, error) {
		return "", errors.New("this is an error")
	}
	commonOpts := &opts.CommonOptions{}
	opts := create.CreateQuickstartOptions{
		CreateProjectOptions: create.CreateProjectOptions{
			ImportOptions: importcmd.ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	// Execute
	err := opts.SetQuickstartPlatform()
	// Evaluate
	assert.NotNil(t, err)
}

func TestCreateQuickstartNewCmdCreateQuickstartFlagsShouldExist(t *testing.T) {
	// Setup
	commonOpts := opts.CommonOptions{}
	cmd := create.NewCmdCreateQuickstart(&commonOpts)
	flagsSet := cmd.Flags()
	type flag struct {
		shorthand string
		value     string
	}
	expected := map[string]flag{
		"organisations":    flag{shorthand: "g", value: "[]"},
		"tag":              flag{shorthand: "t", value: "[]"},
		"owner":            flag{shorthand: "", value: ""},
		"language":         flag{shorthand: "l", value: ""},
		"framework":        flag{shorthand: "", value: ""},
		"git-host":         flag{shorthand: "", value: ""},
		"filter":           flag{shorthand: "f", value: ""},
		"project-name":     flag{shorthand: "p", value: ""},
		"machine-learning": flag{shorthand: "", value: "false"},
		"platform":         flag{shorthand: "", value: "kubernetes"},
		"no-import":        flag{shorthand: "", value: "false"},
		"output-dir":       flag{shorthand: "o", value: ""},
	}
	// Execute & validate
	for expectedKey, expected := range expected {
		flag := flagsSet.Lookup(expectedKey)
		if flag != nil {
			assert.Equal(t, expected.shorthand, flag.Shorthand)
			assert.Equal(t, expected.value, flag.Value.String(), "Default value for "+expectedKey+" is incorrect")
		} else {
			assert.Fail(t, "Could not find the flag "+expectedKey)
		}
	}
}

func TestCreateQuickstartRunCreateGitAuthConfigServiceReturnsError(t *testing.T) {
	// Setup
	orig := opts.CreateGitAuthConfigServiceVar
	defer func() {
		opts.CreateGitAuthConfigServiceVar = orig
	}()
	opts.CreateGitAuthConfigServiceVar = func(o *opts.CommonOptions) (auth.ConfigService, error) {
		return nil, errors.New("It failed")
	}
	opts := create.CreateQuickstartOptions{}
	// Exeucute
	err := opts.Run()
	// Validate
	assert.NotNil(t, err)
}

func TestCreateQuickstartRunJXClientAndDevNamespaceVarReturnsError(t *testing.T) {
	// Setup
	createGitAuthConfigServiceOrig := opts.CreateGitAuthConfigServiceVar
	jxClientAndDevNamespaceOrig := opts.JXClientAndDevNamespaceVar
	defer func() {
		opts.CreateGitAuthConfigServiceVar = createGitAuthConfigServiceOrig
		opts.JXClientAndDevNamespaceVar = jxClientAndDevNamespaceOrig
	}()
	opts.CreateGitAuthConfigServiceVar = func(o *opts.CommonOptions) (auth.ConfigService, error) {
		cs := MockConfigService{}
		return cs, nil
	}
	opts.JXClientAndDevNamespaceVar = func(o *opts.CommonOptions) (versioned.Interface, string, error) {
		return nil, "", errors.New("It failed")
	}
	opts := create.CreateQuickstartOptions{}
	// Execute
	err := opts.Run()
	// Validate
	assert.NotNil(t, err)
}

func TestCreateQuickstartInvokesGetServerlessQuickstartsWhenServerless(t *testing.T) {
	// Setup
	createGitAuthConfigServiceOrig := opts.CreateGitAuthConfigServiceVar
	jxClientAndDevNamespaceOrig := opts.JXClientAndDevNamespaceVar
	getQuickstartLocationsOrig := kube.GetQuickstartLocations
	importCreatedProjectOrig := create.ImportCreatedProjectVar
	defer func() {
		opts.CreateGitAuthConfigServiceVar = createGitAuthConfigServiceOrig
		opts.JXClientAndDevNamespaceVar = jxClientAndDevNamespaceOrig
		kube.GetQuickstartLocations = getQuickstartLocationsOrig
		create.ImportCreatedProjectVar = importCreatedProjectOrig
	}()
	opts.CreateGitAuthConfigServiceVar = func(o *opts.CommonOptions) (auth.ConfigService, error) {
		cs := MockConfigService{}
		return cs, nil
	}
	opts.JXClientAndDevNamespaceVar = func(o *opts.CommonOptions) (versioned.Interface, string, error) {
		return nil, "", nil
	}
	kube.GetQuickstartLocations = func(jxClient versioned.Interface, ns string) ([]v1.QuickStartLocation, error) {
		return nil, nil
	}
	create.ImportCreatedProjectVar = func(outDir string, o *create.CreateProjectOptions) error {
		return nil
	}
	commonOpts := &opts.CommonOptions{
		BatchMode: true,
	}
	opts := create.CreateQuickstartOptions{
		CreateProjectOptions: create.CreateProjectOptions{
			ImportOptions: importcmd.ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	opts.Filter.Platform = "serverless"
	GetServerlessQuickstartsOrig := create.GetServerlessQuickstarts
	defer func() { create.GetServerlessQuickstarts = GetServerlessQuickstartsOrig }()
	actual := false
	create.GetServerlessQuickstarts = func() (*quickstarts.QuickstartModel, error) {
		actual = true
		return quickstarts.NewQuickstartModel(), nil
	}
	// Execute
	opts.Run()
	// Validate
	assert.True(t, actual)
}

func TestCreateQuickstartReturnsErrorWhenGetServerlessQuickstartsFails(t *testing.T) {
	// Setup
	createGitAuthConfigServiceOrig := opts.CreateGitAuthConfigServiceVar
	jxClientAndDevNamespaceOrig := opts.JXClientAndDevNamespaceVar
	getQuickstartLocationsOrig := kube.GetQuickstartLocations
	importCreatedProjectOrig := create.ImportCreatedProjectVar
	GetServerlessQuickstartsOrig := create.GetServerlessQuickstarts
	defer func() {
		opts.CreateGitAuthConfigServiceVar = createGitAuthConfigServiceOrig
		opts.JXClientAndDevNamespaceVar = jxClientAndDevNamespaceOrig
		kube.GetQuickstartLocations = getQuickstartLocationsOrig
		create.ImportCreatedProjectVar = importCreatedProjectOrig
		create.GetServerlessQuickstarts = GetServerlessQuickstartsOrig
	}()
	opts.CreateGitAuthConfigServiceVar = func(o *opts.CommonOptions) (auth.ConfigService, error) {
		cs := MockConfigService{}
		return cs, nil
	}
	opts.JXClientAndDevNamespaceVar = func(o *opts.CommonOptions) (versioned.Interface, string, error) {
		return nil, "", nil
	}
	kube.GetQuickstartLocations = func(jxClient versioned.Interface, ns string) ([]v1.QuickStartLocation, error) {
		return nil, nil
	}
	create.ImportCreatedProjectVar = func(outDir string, o *create.CreateProjectOptions) error {
		return nil
	}
	commonOpts := &opts.CommonOptions{
		BatchMode: true,
	}
	opts := create.CreateQuickstartOptions{
		CreateProjectOptions: create.CreateProjectOptions{
			ImportOptions: importcmd.ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	opts.Filter.Platform = "serverless"
	create.GetServerlessQuickstarts = func() (*quickstarts.QuickstartModel, error) {
		return quickstarts.NewQuickstartModel(), errors.New("this is an error")
	}
	// Execute
	actual := opts.Run()
	// Validate
	assert.Error(t, actual)
}

func TestCreateQuickstartInvokesCreateServerlessQuickstartWhenServerless(t *testing.T) {
	// Setup
	createGitAuthConfigServiceOrig := opts.CreateGitAuthConfigServiceVar
	jxClientAndDevNamespaceOrig := opts.JXClientAndDevNamespaceVar
	getQuickstartLocationsOrig := kube.GetQuickstartLocations
	importCreatedProjectOrig := create.ImportCreatedProjectVar
	getServerlessQuickstartsOrig := create.GetServerlessQuickstarts
	createSurveyVarOrig := quickstarts.CreateSurveyVar
	createServerlessQuickstartOrig := create.CreateServerlessQuickstart
	defer func() {
		opts.CreateGitAuthConfigServiceVar = createGitAuthConfigServiceOrig
		opts.JXClientAndDevNamespaceVar = jxClientAndDevNamespaceOrig
		kube.GetQuickstartLocations = getQuickstartLocationsOrig
		create.ImportCreatedProjectVar = importCreatedProjectOrig
		create.GetServerlessQuickstarts = getServerlessQuickstartsOrig
		quickstarts.CreateSurveyVar = createSurveyVarOrig
		create.CreateServerlessQuickstart = createServerlessQuickstartOrig
	}()
	opts.CreateGitAuthConfigServiceVar = func(o *opts.CommonOptions) (auth.ConfigService, error) {
		cs := MockConfigService{}
		return cs, nil
	}
	opts.JXClientAndDevNamespaceVar = func(o *opts.CommonOptions) (versioned.Interface, string, error) {
		return nil, "", nil
	}
	kube.GetQuickstartLocations = func(jxClient versioned.Interface, ns string) ([]v1.QuickStartLocation, error) {
		return nil, nil
	}
	create.ImportCreatedProjectVar = func(outDir string, o *create.CreateProjectOptions) error {
		return nil
	}
	commonOpts := &opts.CommonOptions{
		BatchMode: true,
	}
	opts := create.CreateQuickstartOptions{
		CreateProjectOptions: create.CreateProjectOptions{
			ImportOptions: importcmd.ImportOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	opts.Filter.Platform = "serverless"
	create.GetServerlessQuickstarts = func() (*quickstarts.QuickstartModel, error) {
		return quickstarts.NewQuickstartModel(), nil
	}
	quickstarts.CreateSurveyVar = func(filter *quickstarts.QuickstartFilter, batchMode bool, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer, model *quickstarts.QuickstartModel) (*quickstarts.QuickstartForm, error) {
		qf := quickstarts.QuickstartForm{
			Quickstart: &quickstarts.Quickstart{},
		}
		return &qf, nil
	}
	actual := false
	create.CreateServerlessQuickstart = func(qf *quickstarts.QuickstartForm, dir string) error {
		actual = true
		return nil
	}
	// Execute
	opts.Run()
	// Validate
	assert.True(t, actual)
}

// Mocks

type MockConfigService struct{}

func (m MockConfigService) Config() *auth.AuthConfig {
	return nil
}

func (m MockConfigService) SetConfig(c *auth.AuthConfig) {}

func (m MockConfigService) LoadConfig() (*auth.AuthConfig, error) {
	return nil, nil
}

func (m MockConfigService) SaveConfig() error {
	return nil
}

func (m MockConfigService) SaveUserAuth(url string, userAuth *auth.UserAuth) error {
	return nil
}

func (m MockConfigService) DeleteServer(url string) error {
	return nil
}
