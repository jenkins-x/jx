package cmd_test

import (
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
)

func TestBuildGitUrl(t *testing.T) {
	sourceRepository := &v1.SourceRepository{
		Spec: v1.SourceRepositorySpec{
			Provider: "https://github.com",
			Org:      "my-org",
			Repo:     "my-repo",
		},
	}
	gitURL := cmd.BuildGitURL(sourceRepository)
	assert.Equal(t, "https://github.com/my-org/my-repo.git", gitURL, "The string should contain a vaild git URL")

	sourceRepository = &v1.SourceRepository{
		Spec: v1.SourceRepositorySpec{
			Provider: "https://invalid",
		},
	}
	gitURL = cmd.BuildGitURL(sourceRepository)
	assert.Equal(t, "None Found", gitURL, "The string should contain the default value")
}
