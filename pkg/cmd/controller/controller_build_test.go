package controller

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"path"
	"strings"
	"testing"
	"time"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/tekton_helpers_test"
	"k8s.io/apimachinery/pkg/runtime"

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
		Namespace:      "test",

		ControllerOptions: ControllerOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}

	testhelpers.ConfigureTestOptionsWithResources(o.CommonOptions,
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
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
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

func TestUpdateForStage(t *testing.T) {
	pod := tekton_helpers_test.AssertLoadSinglePod(t, path.Join("test_data", "controller_build", "update_stage_info"))
	si := &tekton.StageInfo{
		Name:           "ci",
		CreatedTime:    *parseTime(t, "2019-06-07T18:14:19-00:00"),
		FirstStepImage: "gcr.io/abayer-jx-experiment/creds-init:v20190508-91b53326",
		PodName:        "jenkins-x-jx-pr-4135-integratio-42-ci-kdjkp-pod-d964e6",
		TaskRun:        "jenkins-x-jx-pr-4135-integratio-42-ci-kdjkp",
		Task:           "jenkins-x-jx-pr-4135-integratio-ci-42",
		Parents:        []string{},
		Pod:            pod,
	}

	act := &v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "jenkins-x-jx-pr-4135-42",
		},
		Spec: v1.PipelineActivitySpec{
			GitURL:    "https://github.com/jenkins-x/jx",
			GitBranch: "PR-4135",
		},
	}

	updateForStage(si, act)

	containers, _, _ := kube.GetContainersWithStatusAndIsInit(pod)

	assert.Len(t, act.Spec.Steps, 1, "No steps/stages found on activity")
	assert.NotNil(t, act.Spec.Steps[0].Stage, "First step on activity is not a stage")

	stage := act.Spec.Steps[0].Stage

	assert.Equal(t, len(containers), len(stage.Steps), "%d containers found in pod, but %d steps found in stage", len(containers), len(stage.Steps))

	for i, c := range containers {
		step := stage.Steps[i]
		name := strings.Replace(strings.TrimPrefix(c.Name, "build-step-"), "-", " ", -1)
		title := strings.Title(name)
		if title != step.Name {
			// Using t.Errorf instead of an assert here because we want to see all the misordered names, not just the first one.
			t.Errorf("For step %d, expected step with name %s, but found %s", i, title, step.Name)
		}
	}
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
			Name:         "my-repo",
			Organisation: "my-org",
		},
		Commits: []*gits.FakeCommit{
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

func parseTime(t *testing.T, timeString string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, timeString)
	if assert.NoError(t, err, "Failed to parse date %s", timeString) {
		return &parsed
	}
	return nil
}
