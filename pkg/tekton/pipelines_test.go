package tekton_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/tekton"
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
				{Name: "version", Value: "v1"},
				{Name: "build_id", Value: "1"},
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
				{Name: "version", Value: "v1"},
				{Name: "build_id", Value: "1"},
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
				{Name: "version", Value: "v1"},
				{Name: "build_id", Value: "1"},
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
				{Name: "version", Value: "v1"},
				{Name: "build_id", Value: "1"},
			},
		},
		Status: v1alpha1.PipelineRunStatus{
			TaskRuns: taskRunStatusMap,
		},
	}

	assert.True(t, tekton.PipelineRunIsNotPending(pr))
}
