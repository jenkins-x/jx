package opts

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"testing"
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
