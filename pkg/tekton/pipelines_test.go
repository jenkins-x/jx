package tekton_test

import (
	"path"
	"testing"
	"time"

	jxfake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/tekton/tekton_helpers_test"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ns = "jx"
)

func TestPipelineRunIsNotPendingCompletedRun(t *testing.T) {
	now := metav1.Now()
	pr := &v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "PR1",
			Namespace: ns,
			Labels: map[string]string{
				tekton.LabelRepo:    "fakerepo",
				tekton.LabelBranch:  "fakebranch",
				tekton.LabelOwner:   "fakeowner",
				tekton.LabelContext: "fakecontext",
			},
		},
		Spec: v1alpha1.PipelineRunSpec{
			Params: []v1alpha1.Param{
				{
					Name:  "version",
					Value: syntax.StringParamValue("v1"),
				},
				{
					Name:  "build_id",
					Value: syntax.StringParamValue("1"),
				},
			},
		},
		Status: v1alpha1.PipelineRunStatus{
			CompletionTime: &now,
		},
	}

	assert.True(t, tekton.PipelineRunIsNotPending(pr))
}

func TestPipelineRunIsNotPendingRunningSteps(t *testing.T) {
	taskRunStatusMap := make(map[string]*v1alpha1.PipelineRunTaskRunStatus)
	taskRunStatusMap["faketaskrun"] = &v1alpha1.PipelineRunTaskRunStatus{
		Status: &v1alpha1.TaskRunStatus{
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}},
		},
	}

	pr := &v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "PR1",
			Namespace: ns,
			Labels: map[string]string{
				tekton.LabelRepo:    "fakerepo",
				tekton.LabelBranch:  "fakebranch",
				tekton.LabelOwner:   "fakeowner",
				tekton.LabelContext: "fakecontext",
			},
		},
		Spec: v1alpha1.PipelineRunSpec{
			Params: []v1alpha1.Param{
				{Name: "version", Value: syntax.StringParamValue("v1")},
				{Name: "build_id", Value: syntax.StringParamValue("1")},
			},
		},
		Status: v1alpha1.PipelineRunStatus{
			TaskRuns: taskRunStatusMap,
		},
	}

	assert.True(t, tekton.PipelineRunIsNotPending(pr))
}

func TestPipelineRunIsNotPendingWaitingSteps(t *testing.T) {
	taskRunStatusMap := make(map[string]*v1alpha1.PipelineRunTaskRunStatus)
	taskRunStatusMap["faketaskrun"] = &v1alpha1.PipelineRunTaskRunStatus{
		Status: &v1alpha1.TaskRunStatus{
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Message: "Pending",
					},
				},
			}},
		},
	}

	pr := &v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "PR1",
			Namespace: ns,
			Labels: map[string]string{
				tekton.LabelRepo:    "fakerepo",
				tekton.LabelBranch:  "fakebranch",
				tekton.LabelOwner:   "fakeowner",
				tekton.LabelContext: "fakecontext",
			},
		},
		Spec: v1alpha1.PipelineRunSpec{
			Params: []v1alpha1.Param{
				{Name: "version", Value: syntax.StringParamValue("v1")},
				{Name: "build_id", Value: syntax.StringParamValue("1")},
			},
		},
		Status: v1alpha1.PipelineRunStatus{
			TaskRuns: taskRunStatusMap,
		},
	}

	assert.False(t, tekton.PipelineRunIsNotPending(pr))
}

func TestPipelineRunIsNotPendingWaitingStepsInPodInitializing(t *testing.T) {
	taskRunStatusMap := make(map[string]*v1alpha1.PipelineRunTaskRunStatus)
	taskRunStatusMap["faketaskrun"] = &v1alpha1.PipelineRunTaskRunStatus{
		Status: &v1alpha1.TaskRunStatus{
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason: "PodInitializing",
					},
				},
			}},
		},
	}

	pr := &v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "PR1",
			Namespace: ns,
			Labels: map[string]string{
				tekton.LabelRepo:    "fakerepo",
				tekton.LabelBranch:  "fakebranch",
				tekton.LabelOwner:   "fakeowner",
				tekton.LabelContext: "fakecontext",
			},
		},
		Spec: v1alpha1.PipelineRunSpec{
			Params: []v1alpha1.Param{
				{Name: "version", Value: syntax.StringParamValue("v1")},
				{Name: "build_id", Value: syntax.StringParamValue("1")},
			},
		},
		Status: v1alpha1.PipelineRunStatus{
			TaskRuns: taskRunStatusMap,
		},
	}

	assert.True(t, tekton.PipelineRunIsNotPending(pr))
}

func TestGenerateNextBuildNumber(t *testing.T) {
	testCases := []struct {
		name                string
		expectedBuildNumber string
	}{{
		name:                "valid",
		expectedBuildNumber: "309",
	},
		{
			name:                "no_activities",
			expectedBuildNumber: "1",
		},
		{
			name:                "unparseable_build_number",
			expectedBuildNumber: "308",
		}}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			testCaseDir := path.Join("test_data", "next_build_number", tt.name)

			activities := tekton_helpers_test.AssertLoadPipelineActivities(t, testCaseDir)

			commonOpts := opts.NewCommonOptionsWithFactory(fake.NewFakeFactory())
			options := &commonOpts
			testhelpers.ConfigureTestOptions(options, options.Git(), options.Helm())

			tektonClient, ns, err := options.TektonClient()
			assert.NoError(t, err, "There shouldn't be any error getting the fake Tekton Client")

			jxClient := jxfake.NewSimpleClientset(activities)

			repo := &gits.GitRepository{
				Name:         "jx",
				Host:         "github.com",
				Organisation: "jenkins-x",
			}
			nextBuildNumber, err := tekton.GenerateNextBuildNumber(tektonClient, jxClient, ns, repo, "master", 30*time.Second, "release", true)
			assert.NoError(t, err, "There shouldn't be an error getting the next build number")
			assert.Equal(t, tt.expectedBuildNumber, nextBuildNumber)
		})
	}
}
