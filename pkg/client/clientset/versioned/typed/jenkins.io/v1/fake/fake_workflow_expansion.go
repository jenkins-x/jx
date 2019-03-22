package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeWorkflows) PatchUpdate(app *v1.Workflow) (*v1.Workflow, error) {
	return c.Update(app)
}
