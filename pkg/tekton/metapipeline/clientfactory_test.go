package metapipeline

import (
	jxv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"testing"
)

type testTypeAndPullRef struct {
	pipelineKind PipelineKind
	pullRef      PullRef
	expected     string
	errorMessage string
}

func Test_determine_branch_identifier_from_pull_refs(t *testing.T) {
	testCases := []testTypeAndPullRef{
		{
			ReleasePipeline, NewPullRefWithPullRequest("http://foo", "master", "0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815", PullRequestRef{}), "master", "",
		},
		{
			PullRequestPipeline, NewPullRefWithPullRequest("http://foo", "master", "0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815", PullRequestRef{ID: "4554", MergeSHA: "1c313425db5b014271d0d074dd5aac635ffc617e"}), "PR-4554", "",
		},
		{
			PullRequestPipeline, NewPullRef("http://foo", "master", "0967f9ecd7dd2d0acf883c7656c9dc2ad2bf9815"), "", "pullrequest pipeline requested, but no pull requests specified",
		},
	}

	clientFactory := ClientFactory{}

	for _, testCase := range testCases {
		actualBranchIdentifier, err := clientFactory.determineBranchIdentifier(testCase.pipelineKind, testCase.pullRef)
		if testCase.errorMessage == "" {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
			assert.Equal(t, testCase.errorMessage, err.Error())
		}
		assert.Equal(t, testCase.expected, actualBranchIdentifier)
	}
}

func Test_default_version_stream_and_url(t *testing.T) {
	jxClient := fake.NewSimpleClientset()
	client := ClientFactory{
		jxClient: jxClient,
		ns:       "jx",
	}

	url, ref, err := client.versionStreamURLAndRef()
	assert.NoError(t, err)
	assert.Equal(t, config.DefaultVersionsURL, url)
	assert.Equal(t, config.DefaultVersionsRef, ref)
}

func Test_version_stream_and_url_from_team_setting(t *testing.T) {
	var jxObjects []runtime.Object
	expectedUrl := "https://github.com/jenkins-x/my-jenkins-x-versions.git"
	expectedVersion := "v1.0.0"

	ns := "jx"
	devEnv := kube.NewPermanentEnvironment("dev")
	devEnv.Spec.Namespace = ns
	devEnv.Spec.Kind = jxv1.EnvironmentKindTypeDevelopment
	devEnv.Spec.TeamSettings.VersionStreamURL = expectedUrl
	devEnv.Spec.TeamSettings.VersionStreamRef = expectedVersion

	jxObjects = append(jxObjects, devEnv)
	jxClient := fake.NewSimpleClientset(jxObjects...)

	client := ClientFactory{
		jxClient: jxClient,
		ns:       "jx",
	}

	url, ref, err := client.versionStreamURLAndRef()
	assert.NoError(t, err)
	assert.Equal(t, expectedUrl, url)
	assert.Equal(t, expectedVersion, ref)
}

func Test_clone_version_stream_from_ref(t *testing.T) {
	client := ClientFactory{}

	ref := "v1.0.8"
	dir, err := client.cloneVersionStream("https://github.com/jenkins-x/jenkins-x-versions.git", ref)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	assert.NoError(t, err)
	assert.DirExists(t, dir)

	args := []string{"describe", "--tags", "--abbrev=0"}
	cmd := util.Command{
		Dir:  dir,
		Name: "git",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	assert.NoError(t, err)
	assert.Equal(t, ref, output)
}
