package cmd

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

// TODO refactor to encapsulate
func TestGetWorkflow(t *testing.T) {
	if os.Getenv("RUN_UNENCAPSULATED_TESTS") == "true" {
		o := &GetWorkflowOptions{}

		staging := kube.NewPermanentEnvironment("staging")
		production := kube.NewPermanentEnvironment("production")
		staging.Spec.Order = 100
		production.Spec.Order = 200

		myFlowName := "myflow"
		ConfigureTestOptionsWithResources(&o.CommonOptions,
			[]runtime.Object{},
			[]runtime.Object{
				staging,
				production,
				kube.NewPreviewEnvironment("jx-jstrachan-demo96-pr-1"),
				kube.NewPreviewEnvironment("jx-jstrachan-another-pr-3"),
				workflow.CreateWorkflow("jx", myFlowName,
					workflow.CreateWorkflowPromoteStep("a"),
					workflow.CreateWorkflowPromoteStep("b"),
					workflow.CreateWorkflowPromoteStep("c"),
					workflow.CreateWorkflowPromoteStep("d"),
				),
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
					assertPromoteStep(t, &spec.Steps[0], "staging")
				}
				if len(spec.Steps) > 1 {
					assertPromoteStep(t, &spec.Steps[1], "production")
				}
			}
		}

		o.Name = myFlowName
		err = o.Run()
		assert.NoError(t, err)
	} else {
		t.Skip("skipping TestGetWorkflow; RUN_UNENCAPSULATED_TESTS not set")
	}
}

func assertPromoteStep(t *testing.T, step *v1.WorkflowStep, expectedEnvironment string) {
	promote := step.Promote
	assert.True(t, promote != nil, "step is a promote step")

	if promote != nil {
		assert.Equal(t, expectedEnvironment, promote.Environment, "environment name")
	}
}
