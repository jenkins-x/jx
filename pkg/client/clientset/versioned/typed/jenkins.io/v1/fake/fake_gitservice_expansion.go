package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeGitServices) PatchUpdate(app *v1.GitService) (*v1.GitService, error) {
	return c.Update(app)
}
