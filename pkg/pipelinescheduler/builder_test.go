package pipelinescheduler_test

import (
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler/testhelpers"

	"github.com/stretchr/testify/assert"
)

func TestBuildWithEverythingInParent(t *testing.T) {
	child := &v1.SchedulerSpec{
		// Override nothing, everything comes from
	}
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*v1.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent, merged)
}

func TestBuildWithEverythingInChild(t *testing.T) {
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*v1.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, child, merged)
}

func TestBuildWithMergedMerger(t *testing.T) {
	child := testhelpers.CompleteScheduler()
	child.Merger.ContextPolicy = nil
	child.Merger.MergeType = nil
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*v1.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent.Merger.ContextPolicy, merged.Merger.ContextPolicy)
	assert.Equal(t, parent.Merger.MergeType, merged.Merger.MergeType)
	assert.Equal(t, child.Merger.SquashLabel, merged.Merger.SquashLabel)
}
