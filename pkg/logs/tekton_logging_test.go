package logs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/tekton_helpers_test"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonMocks "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kubeMocks "k8s.io/client-go/kubernetes/fake"
)

type TestWriter struct{}

func TestGetTektonPipelinesWithActivePipelineActivityNoData(t *testing.T) {
	jxClient, tektonClient, _, _, ns := getFakeClientsAndNs(t)

	names, paNames, err := GetTektonPipelinesWithActivePipelineActivity(jxClient, tektonClient, ns, []string{}, "")

	assert.NoError(t, err, "There shouldn't be any error obtaining PipelineActivities and PipelineRuns")
	assert.Empty(t, names, "There shouldn't be any returned build names")
	assert.Empty(t, paNames, "There shouldn't be any returned PipelineActivities")
}

func TestGetTektonPipelinesWithActivePipelineActivitySingleBuild(t *testing.T) {
	jxClient, tektonClient, _, _, ns := getFakeClientsAndNs(t)

	_, err := jxClient.JenkinsV1().PipelineActivities(ns).Create(&v1.PipelineActivity{
		ObjectMeta: v12.ObjectMeta{
			Name:      "PA1",
			Namespace: ns,
			Labels: map[string]string{
				v1.LabelRepository: "fakerepo",
				v1.LabelBranch:     "fakebranch",
				v1.LabelOwner:      "fakeowner",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Build:         "1",
			GitBranch:     "fakebranch",
			GitRepository: "fakerepo",
			GitOwner:      "fakeowner",
		},
	})
	assert.NoError(t, err)

	_, err = tektonClient.TektonV1alpha1().PipelineRuns(ns).Create(&v1alpha1.PipelineRun{
		ObjectMeta: v12.ObjectMeta{
			Name:      "PR1",
			Namespace: ns,
			Labels: map[string]string{
				tekton.LabelRepo:    "fakerepo",
				tekton.LabelBranch:  "fakebranch",
				tekton.LabelOwner:   "fakeowner",
				tekton.LabelContext: "fakecontext",
			},
		},
		Spec: v1alpha1.PipelineRunSpec{
			Params: []v1alpha1.Param{
				{Name: "version", Value: "v1"},
				{Name: "build_id", Value: "1"},
			},
		},
	})
	assert.NoError(t, err)

	_, err = tektonClient.TektonV1alpha1().PipelineRuns(ns).Create(&v1alpha1.PipelineRun{
		ObjectMeta: v12.ObjectMeta{
			Name:      "PR2",
			Namespace: ns,
			Labels: map[string]string{
				tekton.LabelBuild:   "2",
				tekton.LabelRepo:    "fakerepo",
				tekton.LabelBranch:  "fakebranch",
				tekton.LabelOwner:   "fakeowner",
				tekton.LabelContext: "tekton",
			},
		},
	})
	assert.NoError(t, err)

	names, paNames, err := GetTektonPipelinesWithActivePipelineActivity(jxClient, tektonClient, ns, []string{}, "fakecontext")

	assert.NoError(t, err, "There shouldn't be any error obtaining PipelineActivities and PipelineRuns")
	assert.Equal(t, "fakeowner/fakerepo/fakebranch #1 fakecontext", names[0], "There should be a match build in the returned names")
	_, exists := paNames[names[0]]
	assert.True(t, exists, "There should be a matching PipelineActivity in the paMap")
	assert.Equal(t, len(names), len(paNames))
}

func TestGetRunningBuildLogsNoBuildPods(t *testing.T) {
	_, tektonClient, kubeClient, _, ns := getFakeClientsAndNs(t)

	pa := &v1.PipelineActivity{
		ObjectMeta: v12.ObjectMeta{
			Name:      "PA1",
			Namespace: ns,
			Labels: map[string]string{
				v1.LabelRepository: "fakerepo",
				v1.LabelBranch:     "fakebranch",
				v1.LabelOwner:      "fakeowner",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Build:         "1",
			GitBranch:     "fakebranch",
			GitRepository: "fakerepo",
			GitOwner:      "fakeowner",
		},
	}

	err := GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1", kubeClient, tektonClient, nil)
	assert.Error(t, err)
	assert.Equal(t, "the build pods for this build have been garbage collected and the log was not found in the long term storage bucket", err.Error())
}

func TestGetRunningBuildLogsWithPipelineRunButNoBuildPods(t *testing.T) {
	testCaseDir := path.Join("test_data")
	_, _, kubeClient, _, ns := getFakeClientsAndNs(t)

	pipelineRun := tekton_helpers_test.AssertLoadSinglePipelineRun(t, testCaseDir)
	tektonClient := tektonMocks.NewSimpleClientset(pipelineRun)

	pa := &v1.PipelineActivity{
		ObjectMeta: v12.ObjectMeta{
			Name:      "PA1",
			Namespace: ns,
			Labels: map[string]string{
				v1.LabelRepository: "fakerepo",
				v1.LabelBranch:     "fakebranch",
				v1.LabelOwner:      "fakeowner",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Build:         "1",
			GitBranch:     "fakebranch",
			GitRepository: "fakerepo",
			GitOwner:      "fakeowner",
		},
	}

	err := GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1", kubeClient, tektonClient, nil)
	assert.Error(t, err)
	assert.Equal(t, "the build pods for this build have been garbage collected and the log was not found in the long term storage bucket", err.Error())
}

func TestGetRunningBuildLogsNoMatchingBuildPods(t *testing.T) {
	testCaseDir := path.Join("test_data")
	_, tektonClient, _, _, ns := getFakeClientsAndNs(t)

	podsList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)
	kubeClient := kubeMocks.NewSimpleClientset(podsList)

	pa := &v1.PipelineActivity{
		ObjectMeta: v12.ObjectMeta{
			Name:      "PA1",
			Namespace: ns,
			Labels: map[string]string{
				v1.LabelRepository: "fakerepo",
				v1.LabelBranch:     "fakebranch",
				v1.LabelOwner:      "fakeowner",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Build:         "1",
			GitBranch:     "fakebranch",
			GitRepository: "fakerepo",
			GitOwner:      "fakeowner",
		},
	}

	err := GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1", kubeClient, tektonClient, nil)
	assert.Error(t, err)
	assert.Equal(t, "the build pods for this build have been garbage collected and the log was not found in the long term storage bucket", err.Error())
}

func TestGetRunningBuildLogsWithMatchingBuildPods(t *testing.T) {
	testCaseDir := path.Join("test_data")
	_, _, _, _, ns := getFakeClientsAndNs(t)

	podsList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)
	pipelineRun := tekton_helpers_test.AssertLoadSinglePipelineRun(t, testCaseDir)
	kubeClient := kubeMocks.NewSimpleClientset(podsList)
	tektonClient := tektonMocks.NewSimpleClientset(pipelineRun)

	pa := &v1.PipelineActivity{
		ObjectMeta: v12.ObjectMeta{
			Name:      "PA1",
			Namespace: ns,
			Labels: map[string]string{
				v1.LabelRepository: "fakerepo",
				v1.LabelBranch:     "fakebranch",
				v1.LabelOwner:      "fakeowner",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Build:         "1",
			GitBranch:     "fakebranch",
			GitRepository: "fakerepo",
			GitOwner:      "fakeowner",
		},
	}

	writer := TestWriter{}
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)

	err := GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1", kubeClient, tektonClient, writer)

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()

	aORb := regexp.MustCompile("Pod logs...")
	n := aORb.FindAllStringIndex(string(outBytes), -1)
	fmt.Println(len(n))

	containers1, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[0])
	containers2, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[1])
	containersNumber := len(containers1) + len(containers2)

	assert.NoError(t, err)
	assert.Equal(t, containersNumber, len(n))
}

func TestGetRunningBuildLogsForLegacyPipelineRunWithMatchingBuildPods(t *testing.T) {
	testCaseDir := path.Join("test_data", "legacy_pipeline_run")
	_, _, _, _, ns := getFakeClientsAndNs(t)

	podsList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)
	pipelineRun := tekton_helpers_test.AssertLoadSinglePipelineRun(t, testCaseDir)
	kubeClient := kubeMocks.NewSimpleClientset(podsList)
	tektonClient := tektonMocks.NewSimpleClientset(pipelineRun)

	pa := &v1.PipelineActivity{
		ObjectMeta: v12.ObjectMeta{
			Name:      "PA1",
			Namespace: ns,
			Labels: map[string]string{
				v1.LabelRepository: "fakerepo",
				v1.LabelBranch:     "fakebranch",
				v1.LabelOwner:      "fakeowner",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Build:         "1",
			GitBranch:     "fakebranch",
			GitRepository: "fakerepo",
			GitOwner:      "fakeowner",
		},
	}

	writer := TestWriter{}
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)

	err := GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1", kubeClient, tektonClient, writer)

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()

	aORb := regexp.MustCompile("Pod logs...")
	n := aORb.FindAllStringIndex(string(outBytes), -1)
	fmt.Println(len(n))

	containers1, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[0])
	containers2, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[1])
	containersNumber := len(containers1) + len(containers2)

	assert.NoError(t, err)
	assert.Equal(t, containersNumber, len(n))
}

func TestStreamPipelinePersistentLogsNotInBucket(t *testing.T) {
	_, _, _, opts, _ := getFakeClientsAndNs(t)
	opts.SkipAuthSecretsMerge = true
	writer := TestWriter{}
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)

	err := StreamPipelinePersistentLogs(writer, "http://nonBucketUrl")
	assert.NoError(t, err)

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()

	assert.Contains(t, string(outBytes), "The build pods for this build have been garbage collected and long term storage bucket configuration wasn't found for this environment")
}

func TestStreamPipelinePersistentLogsInUnsupportedBucketProvider(t *testing.T) {
	_, _, _, opts, _ := getFakeClientsAndNs(t)
	opts.SkipAuthSecretsMerge = true
	writer := TestWriter{}
	r, fakeStdout, _ := os.Pipe()
	log.SetOutput(fakeStdout)

	err := StreamPipelinePersistentLogs(writer, "s3://nonSupportedBucket")
	assert.NoError(t, err)

	fakeStdout.Close()
	outBytes, _ := ioutil.ReadAll(r)
	r.Close()

	assert.Contains(t, string(outBytes), "The provided logsURL scheme is not supported: s3")
}

// Helper method, not supposed to be a test by itself
func getFakeClientsAndNs(t *testing.T) (versioned.Interface, tektonclient.Interface, kubernetes.Interface, opts.CommonOptions, string) {
	commonOpts := opts.NewCommonOptionsWithFactory(fake.NewFakeFactory())
	options := &commonOpts
	testhelpers.ConfigureTestOptions(options, options.Git(), options.Helm())

	jxClient, ns, err := options.JXClientAndDevNamespace()
	assert.NoError(t, err, "There shouldn't be any error getting the fake JXClient and DevEnv")

	tektonClient, _, err := options.TektonClient()
	assert.NoError(t, err, "There shouldn't be any error getting the fake Tekton Client")

	kubeClient, err := options.KubeClient()
	assert.NoError(t, err, "There shouldn't be any error getting the fake Kube Client")

	return jxClient, tektonClient, kubeClient, commonOpts, ns
}

func (w TestWriter) WriteLog(line string) error {
	log.Logger().Info(line)
	return nil
}

func (w TestWriter) StreamLog(ns string, pod *corev1.Pod, container *corev1.Container) error {
	log.Logger().Info("Pod logs...")
	return nil
}
