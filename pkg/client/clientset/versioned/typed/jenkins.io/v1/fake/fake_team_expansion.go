package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeTeams) PatchUpdate(app *v1.Team) (*v1.Team, error) {
	return c.Update(app)
}
