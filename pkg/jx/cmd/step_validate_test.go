package cmd_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	gits_test "github.com/jenkins-x/jx/pkg/gits/mocks"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/stretchr/testify/assert"
)

func TestStepValidate(t *testing.T) {
	t.Parallel()
	AssertValidateWorks(t, &cmd.StepValidateOptions{StepOptions: opts.StepOptions{CommonOptions: &opts.CommonOptions{}}})
	AssertValidateWorks(t, &cmd.StepValidateOptions{StepOptions: opts.StepOptions{CommonOptions: &opts.CommonOptions{}}, MinimumJxVersion: "0.0.1"})

	AssertValidateFails(t, &cmd.StepValidateOptions{StepOptions: opts.StepOptions{CommonOptions: &opts.CommonOptions{}}, MinimumJxVersion: "100.0.1"})

	// lets check the test data has a valid addon
	projectDir := "test_data/project_with_kubeless"
	cfg, fileName, err := config.LoadProjectConfig(projectDir)
	assert.Nil(t, err, "Failed to load project config %s", fileName)
	assert.NotEmpty(t, cfg.Addons, "Failed to find addons in project config %s", fileName)

	AssertValidateFails(t, &cmd.StepValidateOptions{StepOptions: opts.StepOptions{CommonOptions: &opts.CommonOptions{}}, Dir: projectDir})
}

func AssertValidateWorks(t *testing.T, options *cmd.StepValidateOptions) error {
	//options.Out = tests.Output()
	//options.Factory = cmd_mocks.NewMockFactory()
	cmd.ConfigureTestOptions(options.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
	err := options.Run()
	assert.NoError(t, err, "Command failed: %#v", options)
	return err
}

func AssertValidateFails(t *testing.T, options *cmd.StepValidateOptions) error {
	//options.Out = tests.Output()
	//options.Factory = cmd_mocks.NewMockFactory()
	cmd.ConfigureTestOptions(options.CommonOptions, gits_test.NewMockGitter(), helm_test.NewMockHelmer())
	err := options.Run()
	assert.NotNil(t, err, "Command should have failed: %#v", options)
	return err
}
