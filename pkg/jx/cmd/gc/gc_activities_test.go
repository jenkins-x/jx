package gc

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/jx/cmd/testhelpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"testing"
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
	}

	jxClient, ns, err := options.JXClientAndDevNamespace()
	assert.NoError(t, err)
	err = options.ModifyDevEnvironment(func(env *v1.Environment) error {
		env.Spec.TeamSettings.PromotionEngine = jenkinsv1.PromotionEngineProw
		return nil
	})
	assert.NoError(t, err)

	nowMinusThreeDays := time.Now().AddDate(0, 0, -3)
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
			Name: "3",
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

	err = o.Run()
	assert.NoError(t, err)

	activities, err := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
	assert.NoError(t, err)

	assert.Len(t, activities.Items, 2, "One of the activities should've been garbage collected")

	var verifier []bool
	for _, v := range activities.Items {
		if v.BranchName() == "batch" || v.BranchName() == "PR-1" {
			verifier = append(verifier, true)
		}
	}
	assert.Len(t, verifier, 2, "Both PR and Batch builds should've been verified")

}
