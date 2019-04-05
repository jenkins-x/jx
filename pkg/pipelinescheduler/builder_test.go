package pipelinescheduler_test

import (
	"github.com/pborman/uuid"
	"testing"

	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler/testhelpers"

	"github.com/stretchr/testify/assert"
)

func TestBuildWithEverythingInParent(t *testing.T) {
	t.Parallel()
	child := &pipelinescheduler.Scheduler{
		// Override nothing, everything comes from
	}
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent, merged)
}

func TestBuildWithEverythingInChild(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, child, merged)
}

func TestBuildWithSomePropertiesMergedLgtm(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	child.LGTM.ReviewActsAsLgtm = nil
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent.LGTM.ReviewActsAsLgtm, merged.LGTM.ReviewActsAsLgtm)
	assert.Equal(t, child.LGTM.StickyLgtmTeam, merged.LGTM.StickyLgtmTeam)
}

func TestBuildWithLgtmEmptyInChild(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	child.LGTM = &pipelinescheduler.Lgtm{}
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, child.LGTM, merged.LGTM)
}

func TestBuildWithSomePropertiesMergedMerger(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	child.Merger.ContextPolicy = nil
	child.Merger.MergeType = nil
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent.Merger.ContextPolicy, merged.Merger.ContextPolicy)
	assert.Equal(t, parent.Merger.MergeType, merged.Merger.MergeType)
	assert.Equal(t, child.Merger.SquashLabel, merged.Merger.SquashLabel)
}

func TestBuildWithEmptyMerger(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	child.Merger = &pipelinescheduler.Merger{}
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent.Merger, merged.Merger)
}


/*
	Post Submit section
 */
func TestPostSubmitWithEmptyChildPostSubmit(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Postsubmits = &pipelinescheduler.Postsubmits{
		Items: []*pipelinescheduler.Postsubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Postsubmits.Items[0].Name,
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(merged.Postsubmits.Items))
	postSubmit := merged.Postsubmits.Items[0]
	assert.NotNil(t, postSubmit.Branches)
	assert.NotNil(t, postSubmit.Report)
	assert.NotNil(t, postSubmit.Context)
}

func TestPostSubmitMultipleChildrenWithSameNameFound(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Postsubmits = &pipelinescheduler.Postsubmits{
		Items: []*pipelinescheduler.Postsubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Postsubmits.Items[0].Name,
				},
			},
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Postsubmits.Items[0].Name,
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "more than one postsubmit with name")
	assert.Nil(t, merged)
}

func TestPostSubmitWithMergedJobBaseLabels(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Postsubmits = &pipelinescheduler.Postsubmits{
		Items: []*pipelinescheduler.Postsubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Postsubmits.Items[0].Name,
					Labels: testhelpers.PointerToReplaceableMapOfStringString(),
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(merged.Postsubmits.Items))
	postSubmit := merged.Postsubmits.Items[0]
	var labelKey string
	for k := range parent.Presubmits.Items[0].Labels.Items {
		labelKey = k
		break
	}
	assert.Equal(t, postSubmit.Labels.Items[labelKey], parent.Postsubmits.Items[0].Labels.Items[labelKey])
	assert.Len(t, postSubmit.Labels.Items, 2)
}

func TestPostSubmitWithNilJobBaseLabels(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Postsubmits = &pipelinescheduler.Postsubmits{
		Items: []*pipelinescheduler.Postsubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Postsubmits.Items[0].Name,
					Labels: &pipelinescheduler.ReplaceableMapOfStringString{
						Items: nil,
					},
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(merged.Postsubmits.Items))
	postSubmit := merged.Postsubmits.Items[0]
	assert.EqualValues(t, postSubmit.Labels.Items, parent.Postsubmits.Items[0].Labels.Items)
}

func TestPostSubmitApplyBrancherWithEmptyChildPostSubmitBrancher(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Postsubmits = &pipelinescheduler.Postsubmits{
		Items: []*pipelinescheduler.Postsubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Postsubmits.Items[0].Name,
				},
				Brancher: &pipelinescheduler.Brancher{},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(merged.Postsubmits.Items))
	assert.Equal(t, parent.Postsubmits.Items[0].Branches, merged.Postsubmits.Items[0].Brancher.Branches)
	assert.Equal(t, parent.Postsubmits.Items[0].SkipBranches, merged.Postsubmits.Items[0].Brancher.SkipBranches)
}

func TestPostSubmitApplyBrancherWithAppendingChildPostSubmitBrancher(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Postsubmits = &pipelinescheduler.Postsubmits{
		Items: []*pipelinescheduler.Postsubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Postsubmits.Items[0].Name,
				},
				Brancher: &pipelinescheduler.Brancher{
					Branches:     testhelpers.PointerToReplaceableSliceOfStrings(),
					SkipBranches: testhelpers.PointerToReplaceableSliceOfStrings(),
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(merged.Postsubmits.Items))
	assert.Equal(t, 2, len(merged.Postsubmits.Items[0].Brancher.Branches.Items))
	assert.Equal(t, 2, len(merged.Postsubmits.Items[0].Brancher.SkipBranches.Items))
}

/*
	PreSubmit section
 */

func TestPreSubmitWithEmptyChildPreSubmit(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	runIfChanged := ""
	child.Presubmits = &pipelinescheduler.Presubmits{
		Items: []*pipelinescheduler.Presubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Presubmits.Items[0].Name,
				},
				RegexpChangeMatcher: &pipelinescheduler.RegexpChangeMatcher{
					RunIfChanged: &runIfChanged,
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(merged.Presubmits.Items))
	preSubmit := merged.Presubmits.Items[0]
	assert.NotNil(t, preSubmit.Branches)
	assert.NotNil(t, preSubmit.AlwaysRun)
	assert.NotNil(t, preSubmit.ContextPolicy)
	assert.NotNil(t, preSubmit.Query)
}

func TestPreSubmitApplyToQueryWithEmptyChildQuery(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Presubmits = &pipelinescheduler.Presubmits{
		Items: []*pipelinescheduler.Presubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Presubmits.Items[0].Name,
				},
				RegexpChangeMatcher: &pipelinescheduler.RegexpChangeMatcher{

				},
				Query: &pipelinescheduler.Query{

				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	preSubmit := merged.Presubmits.Items[0]
	// TODO: Extract these asserts to method
	assert.NotNil(t, preSubmit.Query.Labels)
	assert.NotNil(t, preSubmit.Query.ExcludedBranches)
	assert.NotNil(t, preSubmit.Query.IncludedBranches)
	assert.NotNil(t, preSubmit.Query.Milestone)
	assert.NotNil(t, preSubmit.Query.MissingLabels)
	assert.NotNil(t, preSubmit.Query.ReviewApprovedRequired)
}

func TestPreSubmitApplyToProtectionPolicyWithAppendingChildPolicy(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Presubmits = &pipelinescheduler.Presubmits{
		Items: []*pipelinescheduler.Presubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Presubmits.Items[0].Name,
				},
				RegexpChangeMatcher: &pipelinescheduler.RegexpChangeMatcher{

				},
				Policy: &pipelinescheduler.ProtectionPolicies{
					Items: map[string]*pipelinescheduler.ProtectionPolicy{
						"policy1": {},
					},
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(merged.Presubmits.Items[0].Policy.Items))
	var protectionPolicyKey string
	for k := range parent.Presubmits.Items[0].Policy.Items {
		protectionPolicyKey = k
		break
	}
	assert.Equal(t, parent.Presubmits.Items[0].Policy.Items[protectionPolicyKey],
		merged.Presubmits.Items[0].Policy.Items[protectionPolicyKey])
}

func TestPreSubmitApplyToRepoContextPolicyWithEmptyChildRepoContextPolicy(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Presubmits = &pipelinescheduler.Presubmits{
		Items: []*pipelinescheduler.Presubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Presubmits.Items[0].Name,
				},
				RegexpChangeMatcher: &pipelinescheduler.RegexpChangeMatcher{},
				ContextPolicy:       &pipelinescheduler.RepoContextPolicy{},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent.Presubmits.Items[0].ContextPolicy.Branches,
		merged.Presubmits.Items[0].ContextPolicy.Branches)
	assert.Equal(t, parent.Presubmits.Items[0].ContextPolicy, merged.Presubmits.Items[0].ContextPolicy)
}

func TestPreSubmitApplyToRepoContextPolicyWithAppendingChildRepoContextPolicy(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Presubmits = &pipelinescheduler.Presubmits{
		Items: []*pipelinescheduler.Presubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Presubmits.Items[0].Name,
				},
				RegexpChangeMatcher: &pipelinescheduler.RegexpChangeMatcher{},
				ContextPolicy: &pipelinescheduler.RepoContextPolicy{
					Branches: &pipelinescheduler.ReplaceableMapOfStringContextPolicy{
						Items: map[string]*pipelinescheduler.ContextPolicy{
							uuid.New(): {},
						},
					},
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.NoError(t, err)
	assert.NotEqual(t, parent.Presubmits.Items[0].ContextPolicy.Branches,
		merged.Presubmits.Items[0].ContextPolicy.Branches)
	assert.Len(t, merged.Presubmits.Items[0].ContextPolicy.Branches.Items, 2)
}

func TestPreSubmitMultipleChildrenWithSameNameFound(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	child.Presubmits = &pipelinescheduler.Presubmits{
		Items: []*pipelinescheduler.Presubmit{
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Presubmits.Items[0].Name,
				},
			},
			{
				JobBase: &pipelinescheduler.JobBase{
					Name: parent.Presubmits.Items[0].Name,
				},
			},
		},
	}

	merged, err := pipelinescheduler.Build([]*pipelinescheduler.Scheduler{parent, child})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "more than one presubmit with name ")
	assert.Nil(t, merged)
}
