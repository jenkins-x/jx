package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeSourceRepositories) PatchUpdate(app *v1.SourceRepository) (*v1.SourceRepository, error) {
	return c.Update(app)
}
