package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeEnvironments) PatchUpdate(app *v1.Environment) (*v1.Environment, error) {
	return c.Update(app)
}
