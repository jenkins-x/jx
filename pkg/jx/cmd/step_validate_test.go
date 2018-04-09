package cmd

import (
	"strings"
	"testing"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestStepValidate(t *testing.T) {
	AssertValidateWorks(t, &StepValidateOptions{MinimumJxVersion: "0.0.1"})

	AssertValidateFails(t, &StepValidateOptions{})
	AssertValidateFails(t, &StepValidateOptions{MinimumJxVersion: "100.0.1"})
}

func AssertValidateWorks(t *testing.T, options *StepValidateOptions) error {
	err := options.Run()
	assert.NoError(t, err, "Command failed: %#v", options)
	return err
}

func AssertValidateFails(t *testing.T, options *StepValidateOptions) error {
	err := options.Run()
	assert.NotNil(t, err, "Command should have failed: %#v", options)
	return err
}

func AssertCommandWorks(t *testing.T, dir string, name string, args ...string) error {
	err := util.RunCommand(dir, name, args...)
	assert.NoError(t, err, "Command failed: %s %s", name, strings.Join(args, ", "))
	return err
}

func AssertCommandFails(t *testing.T, dir string, name string, args ...string) error {
	err := util.RunCommand(dir, name, args...)
	assert.NotNil(t, err, "Command should have failed: %s %s", name, strings.Join(args, ", "))
	return err
}
