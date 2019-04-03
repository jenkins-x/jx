package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDigitSuffix(t *testing.T) {
	testData := map[string]string{
		"nosuffix": "",
		"build1":   "1",
		"build123": "123",
	}

	for input, expected := range testData {
		actual := DigitSuffix(input)
		assert.Equal(t, expected, actual, "digitSuffix for %s", input)
	}
}

func TestCompleteBuildSourceInfo(t *testing.T) {
	o := &ControllerBuildOptions{
		gitHubProvider: gits.NewFakeProvider(getFakeRepository()),
		Namespace: "test",

		ControllerOptions: ControllerOptions{
			CommonOptions: &CommonOptions{},
		},
	}

	ConfigureTestOptionsWithResources(o.CommonOptions,
		[]runtime.Object{
			getGitHubSecret(),
		},
		[]runtime.Object{},
		nil,
		nil,
		nil,
		nil,
	)

	act := &v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta {
			Name: "my-org-my-repo-master-1",
		},
		Spec: v1.PipelineActivitySpec{
			GitURL:    "https://github.com/my-org/my-repo.git",
			GitBranch: "master",
		},
	}

	o.completeBuildSourceInfo(act)

	assert.Equal(t, act.Spec.Author, "john.doe")
	assert.Equal(t, act.Spec.LastCommitMessage, "This is a commit message")

	act = &v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta {
			Name: "my-org-my-repo-pr-4",
		},
		Spec: v1.PipelineActivitySpec{
			GitURL:    "https://github.com/my-org/my-repo.git",
			GitBranch: "PR-4",
		},
	}

	o.completeBuildSourceInfo(act)

	assert.Equal(t, act.Spec.Author, "john.doe.pr")
	assert.Equal(t, act.Spec.PullTitle, "This is the PR title")
}

func getGitHubSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jx-pipeline-git-github-github",
			Namespace: "test",
			Labels: map[string]string{
				kube.LabelCredentialsType: kube.ValueCredentialTypeUsernamePassword,
				kube.LabelCreatedBy:       kube.ValueCreatedByJX,
				kube.LabelKind:            "git",
				kube.LabelServiceKind:     "github",
			},
			Annotations: map[string]string{
				kube.AnnotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
				kube.AnnotationURL:                    "https://github.com",
				kube.AnnotationName:                   "GitHub",
			},
		},
		Data: map[string][]byte{
			"username": []byte("test1"),
			"password": []byte("test1"),
		},
	}
}

func getFakeRepository() *gits.FakeRepository {
	return &gits.FakeRepository{
		Owner: "my-org",
		GitRepo: &gits.GitRepository{
			Name: "my-repo",
			Organisation: "my-org",
		},
		Commits: []*gits.FakeCommit {
			{
				Status: gits.CommitSatusSuccess,
				Commit: &gits.GitCommit{
					SHA:     "ba3ae9923ecc1264ccaa668317fb9276d8b1b5ff",
					Message: "This is a commit message",
					Author: &gits.GitUser{
						Login: "john.doe",
					},
				},
			},
		},
		PullRequests: map[int]*gits.FakePullRequest{
			4: {
				PullRequest: &gits.GitPullRequest{
					Author: &gits.GitUser{
						Login: "john.doe.pr",
					},
					Title: "This is the PR title",
				},
			},
		},
	}
}
