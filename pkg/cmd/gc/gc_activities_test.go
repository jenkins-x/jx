package gc

import (
	"strings"
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
)

func TestGCPipelineActivitiesWithBatchAndPRBuilds(t *testing.T) {
	t.Parallel()

	commonOpts := opts.NewCommonOptionsWithFactory(fake.NewFakeFactory())
	options := &commonOpts
	testhelpers.ConfigureTestOptions(options, options.Git(), options.Helm())

	o := &GCActivitiesOptions{
		CommonOptions:           options,
		PullRequestAgeLimit:     time.Hour * 24 * 30,
		ReleaseAgeLimit:         time.Hour * 48,
		ReleaseHistoryLimit:     5,
		PullRequestHistoryLimit: 2,
		PipelineRunAgeLimit:     time.Hour * 2,
		MetapipelineAgeLimit:    time.Minute * 10,
	}

	jxClient, ns, err := options.JXClientAndDevNamespace()
	assert.NoError(t, err)
	tektonClient, ns, err := options.TektonClient()
	assert.NoError(t, err)
	err = options.ModifyDevEnvironment(func(env *v1.Environment) error {
		env.Spec.TeamSettings.PromotionEngine = jenkinsv1.PromotionEngineProw
		return nil
	})
	assert.NoError(t, err)

	nowMinusThirtyOneDays := time.Now().AddDate(0, 0, -31)
	nowMinusThreeDays := time.Now().AddDate(0, 0, -3)
	nowMinusTwoDays := time.Now().AddDate(0, 0, -2)
	nowMinusOneDay := time.Now().AddDate(0, 0, -1)
	nowMinusOneHour := time.Now().Add(-1 * time.Hour)
	nowMinusThreeHours := time.Now().Add(-3 * time.Hour)
	nowMinusFiveMinutes := time.Now().Add(-5 * time.Minute)
	nowMinusFifteenMinutes := time.Now().Add(-15 * time.Minute)

	_, err = jxClient.JenkinsV1().PipelineActivities(ns).Create(&v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "1",
			Labels: map[string]string{
				v1.LabelBranch: "PR-1",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Pipeline:           "org/project/PR-1",
			CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
		},
	})
	assert.NoError(t, err)

	_, err = jxClient.JenkinsV1().PipelineActivities(ns).Create(&v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "2",
			Labels: map[string]string{
				v1.LabelBranch: "PR-1",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Pipeline: "org/project/PR-1",
			// No completion time, to make sure this doesn't get deleted.
		},
	})
	assert.NoError(t, err)

	_, err = jxClient.JenkinsV1().PipelineActivities(ns).Create(&v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "3",
			Labels: map[string]string{
				v1.LabelBranch: "PR-1",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Pipeline:           "org/project/PR-1",
			CompletedTimestamp: &metav1.Time{Time: nowMinusTwoDays},
		},
	})
	assert.NoError(t, err)

	_, err = jxClient.JenkinsV1().PipelineActivities(ns).Create(&v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "4",
			Labels: map[string]string{
				v1.LabelBranch: "PR-1",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Pipeline:           "org/project/PR-1",
			CompletedTimestamp: &metav1.Time{Time: nowMinusOneDay},
		},
	})
	assert.NoError(t, err)

	// To handle potential weirdness around ordering, make sure that the oldest PR activity is in a random
	// spot in the order.
	_, err = jxClient.JenkinsV1().PipelineActivities(ns).Create(&v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "0",
			Labels: map[string]string{
				v1.LabelBranch: "PR-1",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Pipeline:           "org/project/PR-1",
			CompletedTimestamp: &metav1.Time{Time: nowMinusThirtyOneDays},
		},
	})
	assert.NoError(t, err)

	_, err = jxClient.JenkinsV1().PipelineActivities(ns).Create(&v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "5",
			Labels: map[string]string{
				v1.LabelBranch: "batch",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Pipeline:           "org/project/batch",
			CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
		},
	})
	assert.NoError(t, err)

	_, err = jxClient.JenkinsV1().PipelineActivities(ns).Create(&v1.PipelineActivity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "6",
			Labels: map[string]string{
				v1.LabelBranch: "master",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Pipeline:           "org/project/master",
			CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
		},
	})
	assert.NoError(t, err)

	_, err = tektonClient.TektonV1alpha1().PipelineRuns(ns).Create(&tektonv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "run1",
		},
		Status: tektonv1alpha1.PipelineRunStatus{
			CompletionTime: &metav1.Time{Time: nowMinusThreeHours},
		},
	})
	assert.NoError(t, err)
	_, err = tektonClient.TektonV1alpha1().PipelineRuns(ns).Create(&tektonv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "run2",
		},
		Status: tektonv1alpha1.PipelineRunStatus{
			CompletionTime: &metav1.Time{Time: nowMinusOneHour},
		},
	})
	assert.NoError(t, err)

	taskRunStatuses := make(map[string]*tektonv1alpha1.PipelineRunTaskRunStatus)
	taskRunStatuses["some-task-run"] = &tektonv1alpha1.PipelineRunTaskRunStatus{
		PipelineTaskName: metapipeline.MetaPipelineStageName,
	}

	_, err = tektonClient.TektonV1alpha1().PipelineRuns(ns).Create(&tektonv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "meta1",
		},
		Status: tektonv1alpha1.PipelineRunStatus{
			CompletionTime: &metav1.Time{Time: nowMinusFifteenMinutes},
			TaskRuns: taskRunStatuses,
		},
	})
	assert.NoError(t, err)
	_, err = tektonClient.TektonV1alpha1().PipelineRuns(ns).Create(&tektonv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "meta2",
		},
		Status: tektonv1alpha1.PipelineRunStatus{
			CompletionTime: &metav1.Time{Time: nowMinusFiveMinutes},
			TaskRuns: taskRunStatuses,
		},
	})
	assert.NoError(t, err)

	err = o.Run()
	assert.NoError(t, err)

	activities, err := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
	assert.NoError(t, err)

	assert.Len(t, activities.Items, 4, "Two of the activities should've been garbage collected")

	var verifier []bool
	for _, v := range activities.Items {
		if v.BranchName() == "batch" || v.BranchName() == "PR-1" {
			verifier = append(verifier, true)
		}
	}
	assert.Len(t, verifier, 4, "Both PR and Batch builds should've been verified")

	runs, err := tektonClient.TektonV1alpha1().PipelineRuns(ns).List(metav1.ListOptions{})
	assert.NoError(t, err)

	assert.Len(t, runs.Items, 2, "Two PipelineRuns should've been garbage collected")

	var remainingRun tektonv1alpha1.PipelineRun
	var remainingMeta tektonv1alpha1.PipelineRun

	for _, r := range runs.Items {
		if strings.HasPrefix(r.Name, "run") {
			remainingRun = r
		} else if strings.HasPrefix(r.Name, "meta") {
			remainingMeta = r
		}
	}

	assert.NotNil(t, remainingRun.Status.CompletionTime)
	assert.Equal(t, nowMinusOneHour, remainingRun.Status.CompletionTime.Time, "Expected completion time for remaining PipelineRun of %s, but is %s", nowMinusOneHour, remainingRun.Status.CompletionTime.Time)

	assert.NotNil(t, remainingMeta.Status.CompletionTime)
	assert.Equal(t, nowMinusFiveMinutes, remainingMeta.Status.CompletionTime.Time, "Expected completion time for remaining meta PipelineRun of %s, but is %s", nowMinusFiveMinutes, remainingMeta.Status.CompletionTime.Time)
}
