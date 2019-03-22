package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakeBuildPacks) PatchUpdate(buildPack *v1.BuildPack) (*v1.BuildPack, error) {
	return c.Update(buildPack)
}
