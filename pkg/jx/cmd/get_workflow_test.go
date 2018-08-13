package cmd

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDefaultWorkflow(t *testing.T) {
	o := &GetWorkflowOptions{}

	staging := kube.NewPermanentEnvironment("staging")
	production := kube.NewPermanentEnvironment("production")
	staging.Spec.Order = 100
	production.Spec.Order = 200

	ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			staging,
			production,
			kube.NewPreviewEnvironment("jx-jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment("jx-jstrachan-another-pr-3"),
		},
		gits.NewGitCLI(),
		helm.NewHelmCLI("helm", helm.V2, ""),
	)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)
	if err == nil {
		workflow, err := workflow.GetWorkflow("", jxClient, ns)
		assert.NoError(t, err)
		if err == nil {
			assert.Equal(t, "default", workflow.Name, "name")
			spec := workflow.Spec
			assert.Equal(t, 2, len(spec.Steps), "number of steps")
			if len(spec.Steps) > 0 {
				assertPromoteStep(t, &spec.Steps[0], "staging", false)
			}
			if len(spec.Steps) > 1 {
				assertPromoteStep(t, &spec.Steps[1], "production", false)
			}
		}
	}
}

func assertPromoteStep(t *testing.T, step *v1.WorkflowStep, expectedEnvironment string, expectedParallel bool) {
	promote := step.Promote
	assert.True(t, promote != nil, "step is a promote step")

	if promote != nil {
		assert.Equal(t, expectedEnvironment, promote.Environment, "environment name")
		assert.Equal(t, expectedParallel, promote.Parallel, "parallel")
	}
}
