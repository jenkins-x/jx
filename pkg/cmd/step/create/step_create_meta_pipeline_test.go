package create

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testOptions struct {
	options      []string
	values       []string
	expectError  bool
	errorMessage string
}

func Test_required_options_for_meta_pipeline_creation_need_to_be_passed(t *testing.T) {
	testCases := []testOptions{
		{
			[]string{""}, []string{}, true, "Missing option: --source-url",
		},
		{
			[]string{"SourceURL"}, []string{"https://github.com/jenkins-x/jx.git"}, true, "Missing option: --pull-refs",
		},
		{
			[]string{"SourceURL", "PullRefs"}, []string{"https://github.com/jenkins-x/jx.git", "master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815,4554:1c313425db5b014271d0d074dd5aac635ffc617e"}, false, "",
		},
	}

	for _, data := range testCases {
		o := setTestOptions(data)
		err := o.validateCommandLineFlags()
		if data.expectError {
			assert.Error(t, err, fmt.Sprintf("Error expected for input %v", data))
			assert.Contains(t, err.Error(), data.errorMessage, "unexpected validation error")
		} else {
			assert.NoError(t, err, fmt.Sprintf("No error expected for input %v", data))
		}
	}
}

func setTestOptions(data testOptions) StepCreatePipelineOptions {
	o := StepCreatePipelineOptions{}
	for i, option := range data.options {
		if option != "" {
			obj := reflect.Indirect(reflect.ValueOf(&o.Client))
			obj.FieldByName(option).SetString(data.values[i])
		}
	}
	return o
}
