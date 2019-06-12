package opts

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testCommandName = "foo"
	testFlagName    = "snafu"
)

var (
	cmdUnderTest        *cobra.Command
	commonOptsUnderTest CommonOptions
)

func Test_FlagExplicitlySet_returns_true_if_flag_explicitly_set_to_false(t *testing.T) {
	setupTestCommand()

	err := cmdUnderTest.Flags().Parse([]string{testCommandName, fmt.Sprintf("--%s", testFlagName), "false"})
	assert.NoError(t, err)
	explicit := commonOptsUnderTest.IsFlagExplicitlySet(testFlagName)
	assert.True(t, explicit, "the flag should be explicitly set")
}

func Test_FlagExplicitlySet_returns_true_if_flag_explicitly_set_to_true(t *testing.T) {
	setupTestCommand()

	err := cmdUnderTest.Flags().Parse([]string{testCommandName, fmt.Sprintf("--%s", testFlagName), "true"})
	assert.NoError(t, err)
	explicit := commonOptsUnderTest.IsFlagExplicitlySet(testFlagName)
	assert.True(t, explicit, "the flag should be explicitly set")
}

func Test_FlagExplicitlySet_returns_false_if_flag_is_not_set(t *testing.T) {
	setupTestCommand()

	explicit := commonOptsUnderTest.IsFlagExplicitlySet(testFlagName)
	assert.False(t, explicit, "the flag should not be explicitly set")
}

func Test_FlagExplicitlySet_returns_false_if_flag_is_unkown(t *testing.T) {
	setupTestCommand()

	explicit := commonOptsUnderTest.IsFlagExplicitlySet("fubar")
	assert.False(t, explicit, "the flag should be unknown")
}

func Test_NotifyProgress(t *testing.T) {
	setupTestCommand()

	commonOptsUnderTest.NotifyProgress(LogInfo, "hello %s", "world\n")

	actual := ""
	expectedText := "hello again\n"

	commonOptsUnderTest.NotifyCallback = func(level LogLevel, text string) {
		actual = text
	}

	commonOptsUnderTest.NotifyProgress(LogInfo, expectedText)

	assert.Equal(t, expectedText, actual, "callback receives the log message")
}

func Test_JXNamespace(t *testing.T) {
	setupTestCommand()
	commonOptsUnderTest.SetFactory(clients.NewFactory())

	kubeClient, ns, err := commonOptsUnderTest.KubeClientAndNamespace()
	assert.NoError(t, err, "Failed to create kube client")

	if err == nil {
		resource, err := kubeClient.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
		assert.NoError(t, err, "Failed to query namespace")
		if err == nil {
			log.Logger().Warnf("Found namespace %#v", resource)
		}
	}

	_, err = commonOptsUnderTest.CreateGitAuthConfigService()
	assert.NoError(t, err, "Failed to create GitAuthConfigService")
}

func setupTestCommand() {
	var flag bool
	cmdUnderTest = &cobra.Command{
		Use:   testCommandName,
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {
			// noop
		},
	}
	cmdUnderTest.Flags().BoolVar(&flag, testFlagName, false, "")

	commonOptsUnderTest = CommonOptions{}
	commonOptsUnderTest.Cmd = cmdUnderTest
}
