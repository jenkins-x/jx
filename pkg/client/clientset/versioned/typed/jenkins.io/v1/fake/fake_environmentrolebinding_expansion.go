package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeEnvironmentRoleBindings) PatchUpdate(app *v1.EnvironmentRoleBinding) (*v1.EnvironmentRoleBinding, error) {
	return c.Update(app)
}
