// +build unit

package get_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/jenkins-x/jx/v2/pkg/cmd/get"
	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/v2/pkg/tekton"
	"github.com/jenkins-x/jx/v2/pkg/tekton/syntax"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testDevNameSpace = "jx-test"

func pipelineRun(ns, repo, branch, owner, context string, now metav1.Time) *tektonv1alpha1.PipelineRun {
	return &v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "PR1",
			Namespace: ns,
			Labels: map[string]string{
				tekton.LabelRepo:    repo,
				tekton.LabelBranch:  branch,
				tekton.LabelOwner:   owner,
				tekton.LabelContext: context,
			},
		},
		Spec: v1alpha1.PipelineRunSpec{
			Params: []v1alpha1.Param{
				{
					Name:  "version",
					Value: syntax.StringParamValue("v1"),
				},
				{
					Name:  "build_id",
					Value: syntax.StringParamValue("1"),
				},
			},
		},
		Status: v1alpha1.PipelineRunStatus{
			PipelineRunStatusFields: tektonv1beta1.PipelineRunStatusFields{
				CompletionTime: &now,
			},
		},
	}
}

var pipelineCases = []struct {
	desc      string
	namespace string
	repo      string
	branch    string
	owner     string
	context   string
}{
	{"", testDevNameSpace, "testRepo", "testBranch", "testOwner", "testContext"},
	{"", testDevNameSpace, "testRepo", "testBranch", "testOwner", ""},
}

func TestExecuteGetPipelines(t *testing.T) {
	for _, v := range pipelineCases {
		t.Run(v.desc, func(t *testing.T) {
			// fakeout the output for the tests
			out := &testhelpers.FakeOut{}
			commonOpts := opts.NewCommonOptionsWithTerm(clients.NewFactory(), os.Stdin, out, os.Stderr)

			// Set batchmode to true for tests
			commonOpts.BatchMode = true

			// Set dev namespace
			commonOpts.SetDevNamespace(v.namespace)

			// Fake tekton client
			client := fake.NewSimpleClientset(pipelineRun(v.namespace, v.repo, v.branch, v.owner, v.context, metav1.Now()))

			commonOpts.SetTektonClient(client)
			command := get.NewCmdGetPipeline(commonOpts)
			err := command.Execute()

			// Execution should not error out
			assert.NoError(t, err, "execute get pipelines")
		})
	}

}
