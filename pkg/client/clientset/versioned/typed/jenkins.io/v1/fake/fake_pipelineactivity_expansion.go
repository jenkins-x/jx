package fake

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
)

func (c *FakePipelineActivities) PatchUpdate(activity *v1.PipelineActivity) (*v1.PipelineActivity, error) {
	return c.Update(activity)
}
