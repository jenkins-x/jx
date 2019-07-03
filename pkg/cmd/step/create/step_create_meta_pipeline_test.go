package create

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type testOptions struct {
	options      []string
	values       []string
	expectError  bool
	errorMessage string
}

type testPullRef struct {
	pullRef  string
	expected string
}

func Test_required_options_for_meta_pipeline_creation_need_to_be_passed(t *testing.T) {
	testCases := []testOptions{
		{
			[]string{""}, []string{}, true, "Missing option: --source-url",
		},
		{
			[]string{"SourceURL"}, []string{"https://github.com/jenkins-x/jx.git"}, true, "Missing option: --job",
		},
		{
			[]string{"SourceURL", "Job"}, []string{"https://github.com/jenkins-x/jx.git", "5108a21c-9d6b-11e9-b835-9a460f5ab90f"}, true, "Missing option: --pull-refs",
		},
		{
			[]string{"SourceURL", "Job", "PullRefs"}, []string{"https://github.com/jenkins-x/jx.git", "master", "master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815,4554:1c313425db5b014271d0d074dd5aac635ffc617e"}, false, "",
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

func Test_determine_branch_identifier_from_pull_refs(t *testing.T) {
	testCases := []testPullRef{
		{
			"master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815", "master",
		},
		{
			"feature-1:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815", "feature-1",
		},
		{
			"master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815,4554:1c313425db5b014271d0d074dd5aac635ffc617e", "PR-4554",
		},
		{
			"master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815,4554:1c313425db5b014271d0d074dd5aac635ffc617e,5555:1c313425db8b014271d0d074dd5aac635ffc617e", "batch",
		},
	}

	option := StepCreatePipelineOptions{}

	for _, testCase := range testCases {
		pullRefs, err := prow.ParsePullRefs(testCase.pullRef)
		assert.NoError(t, err)
		actualBranchName := option.determineBranchIdentifier(*pullRefs)
		assert.Equal(t, testCase.expected, actualBranchName)
	}
}

func Test_determine_pipeline_kind(t *testing.T) {
	testCases := []testPullRef{
		{
			"master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815", jenkinsfile.PipelineKindRelease,
		},
		{
			"master:0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815,4554:1c313425db5b014271d0d074dd5aac635ffc617e", jenkinsfile.PipelineKindPullRequest,
		},
	}

	option := StepCreatePipelineOptions{}

	for _, testCase := range testCases {
		pullRefs, err := prow.ParsePullRefs(testCase.pullRef)
		assert.NoError(t, err)
		actualPipelineKind := option.determinePipelineKind(*pullRefs)
		assert.Equal(t, testCase.expected, actualPipelineKind)
	}
}

func setTestOptions(data testOptions) StepCreatePipelineOptions {
	o := StepCreatePipelineOptions{}
	for i, option := range data.options {
		if option != "" {
			obj := reflect.Indirect(reflect.ValueOf(&o))
			obj.FieldByName(option).SetString(data.values[i])
		}
	}
	return o
}
