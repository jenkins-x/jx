package cmd

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestSequentialWorkflow(t *testing.T) {
	o := &ControllerWorkflowOptions{
		NoWatch: true,
	}

	staging := kube.NewPermanentEnvironmentWithGit("staging", "https://github.com/jstrachan/environment-jstrachan-test1-staging.git")
	production := kube.NewPermanentEnvironmentWithGit("production", "https://github.com/jstrachan/environment-jstrachan-test1-production.git")
	staging.Spec.Order = 100
	production.Spec.Order = 200

	myFlowName := "myflow"

	step1 := workflow.CreateWorkflowPromoteStep("staging", false)
	step2 := workflow.CreateWorkflowPromoteStep("production", false, step1)

	ConfigureTestOptionsWithResources(&o.CommonOptions,
		[]runtime.Object{},
		[]runtime.Object{
			staging,
			production,
			kube.NewPreviewEnvironment("jx-jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment("jx-jstrachan-another-pr-3"),
			workflow.CreateWorkflow("jx", myFlowName,
				step1,
				step2,
			),
		},
		gits.NewGitCLI(),
		helm.NewHelmCLI("helm", helm.V2, ""),
	)
	o.git = &gits.GitFake{}

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

	_, err = createTestPipelineActivity(jxClient, ns, "jstrachan", "myrepo", "master", "1", myFlowName)
	assert.NoError(t, err)
	if err == nil {
		err = o.Run()
		assert.NoError(t, err)
	}
}

func createTestPipelineActivity(jxClient versioned.Interface, ns string, folder string, repo string, branch string, build string, workflow string) (*v1.PipelineActivity, error) {
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	key := &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:     folder + "-" + repo + "-" + branch + "-" + build,
			Pipeline: folder + "/" + repo + "/" + branch,
			Build:    build,
		},
	}
	a, _, err := key.GetOrCreate(activities)
	version := "1.0.1"
	a.Spec.GitOwner = folder
	a.Spec.GitRepository = repo
	a.Spec.Version = version
	a.Spec.Workflow = workflow
	_, err = activities.Update(a)
	return a, err
}
