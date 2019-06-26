package create

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type testData struct {
	options      []string
	values       []string
	expectError  bool
	errorMessage string
}

func Test_required_options_for_meta_pipeline_creation_need_to_be_passed(t *testing.T) {
	testCases := []testData{
		{
			[]string{""}, []string{}, true, "Missing option: --source-url",
		},
		{
			[]string{"SourceURL"}, []string{"https://github.com/jenkins-x/jx.git"}, true, "Missing option: --branch",
		},
		{
			[]string{"SourceURL", "Branch"}, []string{"https://github.com/jenkins-x/jx.git", "master"}, true, "Missing option: --kind",
		},
		{
			[]string{"SourceURL", "Branch", "PipelineKind"}, []string{"https://github.com/jenkins-x/jx.git", "master", "release"}, true, "Missing option: --job",
		},
		{
			[]string{"SourceURL"}, []string{"foo"}, true, "unable to determine needed git info",
		},
		{
			[]string{"SourceURL", "Branch", "PipelineKind", "Job"}, []string{"https://github.com/jenkins-x/jx.git", "master", "release", "job-id"}, false, "",
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

func setTestOptions(data testData) StepCreatePipelineOptions {
	o := StepCreatePipelineOptions{}
	for i, option := range data.options {
		if option != "" {
			obj := reflect.Indirect(reflect.ValueOf(&o))
			obj.FieldByName(option).SetString(data.values[i])
		}
	}
	return o
}
