package get

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/acarl005/stripansi"
	jxfake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tekton/tekton_helpers_test"
	"github.com/stretchr/testify/assert"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"k8s.io/apimachinery/pkg/runtime"
	kube_mocks "k8s.io/client-go/kubernetes/fake"
)

func TestWithMetapipeline(t *testing.T) {
	testCaseDir := path.Join("test_data", "get_build_logs", "with-metapipeline")

	structures := tekton_helpers_test.AssertLoadPipelineStructures(t, testCaseDir)
	jxClient := jxfake.NewSimpleClientset(structures)

	tektonObjects := []runtime.Object{tekton_helpers_test.AssertLoadPipelineRuns(t, testCaseDir), tekton_helpers_test.AssertLoadPipelines(t, testCaseDir)}
	tektonObjects = append(tektonObjects, tekton_helpers_test.AssertLoadTasks(t, testCaseDir))
	tektonObjects = append(tektonObjects, tekton_helpers_test.AssertLoadTaskRuns(t, testCaseDir))
	tektonClient := tektonfake.NewSimpleClientset(tektonObjects...)

	podList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)
	kubeClient := kube_mocks.NewSimpleClientset(podList)

	ns := "jx"

	o := &GetBuildLogsOptions{
		GetOptions: GetOptions{
			CommonOptions: &opts.CommonOptions{
				BatchMode: true,
			},
		},
	}

	names, defaultName, pipelineMap, err := o.loadPipelines(kubeClient, tektonClient, jxClient, ns)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(names))
	assert.Equal(t, "", defaultName)

	pipelineName := names[0]
	assert.Equal(t, "abayer/js-test-repo/build-pack #8", pipelineName)

	assert.Equal(t, 2, len(pipelineMap[pipelineName]))

	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)
	o.CommonOptions.Out = fakeStdout
	o.Args = []string{pipelineName}

	err = o.getProwBuildLog(kubeClient, tektonClient, jxClient, ns, true)
	assert.NoError(t, err)

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()
	output := stripansi.Strip(string(outBytes))
	assert.Contains(t, output, "stage app extension")
	assert.Contains(t, output, "stage from build pack")
}
