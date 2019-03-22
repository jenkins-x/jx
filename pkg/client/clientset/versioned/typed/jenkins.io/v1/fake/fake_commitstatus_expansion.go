package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeCommitStatuses) PatchUpdate(app *v1.CommitStatus) (*v1.CommitStatus, error) {
	return c.Update(app)
}
