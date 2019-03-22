package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeApps) PatchUpdate(app *v1.App) (*v1.App, error) {
	return c.Update(app)
}
