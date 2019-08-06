package metapipeline

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/stretchr/testify/assert"
)

type testPullRef struct {
	pullRef  string
	expected string
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

	option := Client{}

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

	option := Client{}

	for _, testCase := range testCases {
		pullRefs, err := prow.ParsePullRefs(testCase.pullRef)
		assert.NoError(t, err)
		actualPipelineKind := option.determinePipelineKind(*pullRefs)
		assert.Equal(t, testCase.expected, actualPipelineKind)
	}
}
