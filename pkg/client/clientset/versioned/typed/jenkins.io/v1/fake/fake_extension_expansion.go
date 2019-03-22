package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeExtensions) PatchUpdate(app *v1.Extension) (*v1.Extension, error) {
	return c.Update(app)
}
