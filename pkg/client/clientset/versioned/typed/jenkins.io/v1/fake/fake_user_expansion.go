package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeUsers) PatchUpdate(app *v1.User) (*v1.User, error) {
	return c.Update(app)
}
