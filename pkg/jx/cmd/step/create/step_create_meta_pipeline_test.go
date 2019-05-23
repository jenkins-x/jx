package create

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type testData struct {
	options     []string
	values      []string
	expectError bool
}

func Test_required_options_for_meta_pipeline_creation_need_to_be_passed(t *testing.T) {
	testCases := []testData{
		{
			[]string{""}, []string{}, true,
		},
		{
			[]string{"GitCloneURL"}, []string{"https://github.com/jenkins-x/jx.git"}, true,
		},
		{
			[]string{"GitCloneURL", "Branch"}, []string{"https://github.com/jenkins-x/jx.git", "master"}, true,
		},
		{
			[]string{"GitCloneURL", "Branch", "PipelineKind"}, []string{"https://github.com/jenkins-x/jx.git", "master", "release"}, false,
		},
		{
			[]string{"GitCloneURL", "Branch", "PipelineKind"}, []string{"foo", "master", "release"}, true,
		},
		{
			[]string{"GitCloneURL", "Branch", "PipelineKind"}, []string{"https://github.com/jenkins-x/jx.git", "master", "foo"}, true,
		},
	}

	for _, data := range testCases {
		o := setTestOptions(data)
		err := o.validateCommandLineFlags()
		if data.expectError {
			assert.Error(t, err, fmt.Sprintf("Error expected for input %v", data))
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
