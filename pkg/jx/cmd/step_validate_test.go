package cmd

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
)

func TestStepValidate(t *testing.T) {
	t.Parallel()
	AssertValidateWorks(t, &StepValidateOptions{})
	AssertValidateWorks(t, &StepValidateOptions{MinimumJxVersion: "0.0.1"})

	AssertValidateFails(t, &StepValidateOptions{MinimumJxVersion: "100.0.1"})

	// lets check the test data has a valid addon
	projectDir := "test_data/project_with_kubeless"
	cfg, fileName, err := config.LoadProjectConfig(projectDir)
	assert.Nil(t, err, "Failed to load project config %s", fileName)
	assert.NotEmpty(t, cfg.Addons, "Failed to find addons in project config %s", fileName)

	AssertValidateFails(t, &StepValidateOptions{Dir: projectDir})
}

func AssertValidateWorks(t *testing.T, options *StepValidateOptions) error {
	options.Out = tests.Output()
	err := options.Run()
	assert.NoError(t, err, "Command failed: %#v", options)
	return err
}

func AssertValidateFails(t *testing.T, options *StepValidateOptions) error {
	options.Out = tests.Output()
	err := options.Run()
	assert.NotNil(t, err, "Command should have failed: %#v", options)
	return err
}
