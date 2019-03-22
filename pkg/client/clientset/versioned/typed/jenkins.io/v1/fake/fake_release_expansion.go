package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeReleases) PatchUpdate(app *v1.Release) (*v1.Release, error) {
	return c.Update(app)
}
