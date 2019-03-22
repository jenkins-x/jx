package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeFacts) PatchUpdate(app *v1.Fact) (*v1.Fact, error) {
	return c.Update(app)
}
