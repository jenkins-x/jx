package verify_test

import (
	"os"
	"path"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/step/verify"
	"github.com/stretchr/testify/assert"
)

func createStepVerifyValuesOptions(test string) *verify.StepVerifyValuesOptions {
	dir := path.Join("test_data", "verify_values", test)
	options := verify.StepVerifyValuesOptions{
		SchemaFile:      path.Join(dir, "schema.json"),
		RequirementsDir: dir,
		ValuesFile:      path.Join(dir, "values.yaml"),
	}
	commonOpts := opts.NewCommonOptionsWithFactory(nil)
	commonOpts.Out = os.Stdout
	commonOpts.Err = os.Stderr
	options.CommonOptions = &commonOpts
	return &options
}

func TestStepVerifyValuesWithValidValues(t *testing.T) {
	t.Parallel()

	options := createStepVerifyValuesOptions("valid_values")
	err := options.Run()

	assert.NoError(t, err, "Command failed: %v", options)
}

func TestStepVerifyValuesWithWarningValues(t *testing.T) {
	t.Parallel()

	options := createStepVerifyValuesOptions("warning_values")
	err := options.Run()

	assert.NoError(t, err, "Command failed: %v", options)
}

func TestStepVerifyValuesWithErrorValues(t *testing.T) {
	t.Parallel()

	options := createStepVerifyValuesOptions("error_values")
	err := options.Run()

	assert.Error(t, err, "Command didn't fail: %v", options)
}
