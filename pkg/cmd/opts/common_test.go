package opts_test

import (
	"fmt"
	"testing"

	"io/ioutil"
	"os"
	"path"

	"github.com/jenkins-x/jx/pkg/auth"
	clients_test "github.com/jenkins-x/jx/pkg/cmd/clients/mocks"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/gits"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	testCommandName = "foo"
	testFlagName    = "snafu"
)

type TestFlags struct {
	Snafu      bool       `mapstructure:"snafu"`
	ChildFlags ChildFlags `mapstructure:"children"`
}

type ChildFlags struct {
	Child string `mapstructure:"child"`
}

func Test_FlagExplicitlySet_returns_true_if_flag_explicitly_set_to_false(t *testing.T) {
	cmdUnderTest, commonOptsUnderTest := setupTestCommand(t)

	err := cmdUnderTest.Flags().Parse([]string{testCommandName, fmt.Sprintf("--%s", testFlagName), "false"})
	assert.NoError(t, err)
	explicit := commonOptsUnderTest.IsFlagExplicitlySet(testFlagName)
	assert.True(t, explicit, "the flag should be explicitly set")
}

func Test_FlagExplicitlySet_returns_true_if_flag_explicitly_set_to_true(t *testing.T) {
	cmdUnderTest, commonOptsUnderTest := setupTestCommand(t)

	err := cmdUnderTest.Flags().Parse([]string{testCommandName, fmt.Sprintf("--%s", testFlagName), "true"})
	assert.NoError(t, err)
	explicit := commonOptsUnderTest.IsFlagExplicitlySet(testFlagName)
	assert.True(t, explicit, "the flag should be explicitly set")
}

func Test_FlagExplicitlySet_returns_false_if_flag_is_not_set(t *testing.T) {
	_, commonOptsUnderTest := setupTestCommand(t)

	explicit := commonOptsUnderTest.IsFlagExplicitlySet(testFlagName)
	assert.False(t, explicit, "the flag should not be explicitly set")
}

func Test_FlagExplicitlySet_returns_false_if_flag_is_unknown(t *testing.T) {
	_, commonOptsUnderTest := setupTestCommand(t)

	explicit := commonOptsUnderTest.IsFlagExplicitlySet("fubar")
	assert.False(t, explicit, "the flag should be unknown")
}

func Test_NotifyProgress(t *testing.T) {
	_, commonOptsUnderTest := setupTestCommand(t)

	commonOptsUnderTest.NotifyProgress(opts.LogInfo, "hello %s", "world\n")

	actual := ""
	expectedText := "hello again\n"

	commonOptsUnderTest.NotifyCallback = func(level opts.LogLevel, text string) {
		actual = text
	}

	commonOptsUnderTest.NotifyProgress(opts.LogInfo, expectedText)

	assert.Equal(t, expectedText, actual, "callback receives the log message")
}

func Test_JXNamespace(t *testing.T) {
	_, commonOptsUnderTest := setupTestCommand(t)

	kubeClient, ns, err := commonOptsUnderTest.KubeClientAndNamespace()
	assert.NoError(t, err, "Failed to create kube client")

	if err == nil {
		resource, err := kubeClient.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
		assert.NoError(t, err, "Failed to query namespace")
		if err == nil {
			log.Logger().Warnf("Found namespace %#v", resource)
		}
	}

	_, err = commonOptsUnderTest.CreateGitConfigService()
	assert.NoError(t, err, "Failed to create GitAuthConfigService")
}

func Test_GetConfiguration(t *testing.T) {
	_, commonOptsUnderTest := setupTestCommand(t)

	fileContent := fmt.Sprintf("%s: %t\n", testFlagName, true)
	configFile, removeTmp := setupTestConfig(t, fileContent)

	defer removeTmp(configFile)

	commonOptsUnderTest.ConfigFile = configFile

	testFlags := TestFlags{}
	err := commonOptsUnderTest.GetConfiguration(&testFlags)
	assert.NoError(t, err, "Failed to GetConfiguration")

	assert.Equal(t, true, testFlags.Snafu)
}

func Test_configExists_child(t *testing.T) {
	_, commonOptsUnderTest := setupTestCommand(t)

	valuesYaml := fmt.Sprintf("children:\n  child: foo")
	configFile, removeTmp := setupTestConfig(t, valuesYaml)

	defer removeTmp(configFile)

	assert.True(t, commonOptsUnderTest.ConfigExists("children", "child"))
}

func Test_configExists_no_path(t *testing.T) {
	_, commonOptsUnderTest := setupTestCommand(t)

	valuesYaml := fmt.Sprintf("snafu: true")
	configFile, removeTmp := setupTestConfig(t, valuesYaml)

	defer removeTmp(configFile)

	assert.True(t, commonOptsUnderTest.ConfigExists("", "snafu"))
}

func Test_configNotExists(t *testing.T) {
	_, commonOptsUnderTest := setupTestCommand(t)

	valuesYaml := fmt.Sprintf("children:\n  child: foo")
	configFile, removeTmp := setupTestConfig(t, valuesYaml)

	defer removeTmp(configFile)

	assert.False(t, commonOptsUnderTest.ConfigExists("children", "son"))
}

func setupTestConfig(t *testing.T, config string) (string, func(string)) {
	_, commonOptsUnderTest := setupTestCommand(t)

	tmpDir, err := ioutil.TempDir("", "")
	require.Nil(t, err, "Failed creating tmp dir")
	configFile := path.Join(tmpDir, "config.yaml")
	err = ioutil.WriteFile(configFile, []byte(config), 0640)
	require.Nil(t, err, "Failed writing config yaml file")

	commonOptsUnderTest.ConfigFile = configFile

	testFlags := TestFlags{}
	err = commonOptsUnderTest.GetConfiguration(&testFlags)
	assert.NoError(t, err, "Failed to GetConfiguration")

	removeAllFunc := func(configFile string) {
		_ = os.RemoveAll(configFile)
	}
	return configFile, removeAllFunc
}

func setupTestCommand(t *testing.T) (*cobra.Command, *opts.CommonOptions) {
	var flag bool
	cmd := &cobra.Command{
		Use:   testCommandName,
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {
			// noop
		},
	}
	cmd.Flags().BoolVar(&flag, testFlagName, false, "")
	_ = viper.BindPFlag(testFlagName, cmd.Flags().Lookup(testFlagName))

	mockFactory := clients_test.NewMockFactory()
	commonOpts := opts.NewCommonOptionsWithFactory(mockFactory)
	mockHelmer := helm_test.NewMockHelmer()
	installerMock := resources_test.NewMockInstaller()
	server := auth.Server{
		URL: "https://github.com",
		Users: []auth.User{
			{
				Username: "test",
				ApiToken: "test",
			},
		},
		Name:        "GitHub",
		Kind:        "github",
		CurrentUser: "test",
	}
	config := auth.Config{
		Servers:       []auth.Server{server},
		CurrentServer: server.URL,
	}
	configSvc, err := auth.NewMemConfigService(config)
	if err != nil {
		t.Fatal("failed to create auth config service")
	}
	gitProvider, err := gits.NewFakeProvider(server, &gits.FakeRepository{
		Owner: "test",
		GitRepo: &gits.GitRepository{
			Name: "test",
		},
	})
	if err != nil {
		t.Fatal("failed to create git provider")
	}
	testhelpers.ConfigureTestOptionsWithResources(&commonOpts,
		[]runtime.Object{},
		[]runtime.Object{},
		configSvc,
		gits.NewGitFake(server),
		gitProvider,
		mockHelmer,
		installerMock,
	)

	commonOpts.Cmd = cmd

	return cmd, &commonOpts
}
