package fake

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

func (c *FakePipelineStructures) PatchUpdate(app *v1.PipelineStructure) (*v1.PipelineStructure, error) {
	return c.Update(app)
}
