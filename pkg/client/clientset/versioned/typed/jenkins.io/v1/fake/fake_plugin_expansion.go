package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakePlugins) PatchUpdate(app *v1.Plugin) (*v1.Plugin, error) {
	return c.Update(app)
}
