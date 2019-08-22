package logs

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"regexp"
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	jxfake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
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

type TestWriter struct {
	StreamLinesLogged []string
	SingleLinesLogged []string
}

func TestGetTektonPipelinesWithActivePipelineActivityNoData(t *testing.T) {
	jxClient, tektonClient, _, _, ns := getFakeClientsAndNs(t)
	tl := TektonLogger{
		JXClient:     jxClient,
		TektonClient: tektonClient,
		Namespace:    ns,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}
	names, paNames, err := tl.GetTektonPipelinesWithActivePipelineActivity([]string{}, "")

	assert.NoError(t, err, "There shouldn't be any error obtaining PipelineActivities and PipelineRuns")
	assert.Empty(t, names, "There shouldn't be any returned build names")
	assert.Empty(t, paNames, "There shouldn't be any returned PipelineActivities")
}

func TestGetTektonPipelinesWithActivePipelineActivitySingleBuild(t *testing.T) {
	jxClient, tektonClient, _, _, ns := getFakeClientsAndNs(t)

	tl := TektonLogger{
		JXClient:     jxClient,
		TektonClient: tektonClient,
		Namespace:    ns,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}

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

	taskRunStatusMap := make(map[string]*v1alpha1.PipelineRunTaskRunStatus)
	taskRunStatusMap["faketaskrun"] = &v1alpha1.PipelineRunTaskRunStatus{
		Status: &v1alpha1.TaskRunStatus{
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}},
		},
	}

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
		Status: v1alpha1.PipelineRunStatus{
			TaskRuns: taskRunStatusMap,
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

	names, paNames, err := tl.GetTektonPipelinesWithActivePipelineActivity([]string{}, "fakecontext")

	assert.NoError(t, err, "There shouldn't be any error obtaining PipelineActivities and PipelineRuns")
	assert.Equal(t, "fakeowner/fakerepo/fakebranch #1 fakecontext", names[0], "There should be a match build in the returned names")
	_, exists := paNames[names[0]]
	assert.True(t, exists, "There should be a matching PipelineActivity in the paMap")
	assert.Equal(t, len(names), len(paNames))
}

func TestGetTektonPipelinesWithActivePipelineActivityOnlyWaitingStep(t *testing.T) {
	jxClient, tektonClient, _, _, ns := getFakeClientsAndNs(t)
	tl := TektonLogger{
		JXClient:     jxClient,
		TektonClient: tektonClient,
		Namespace:    ns,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}

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

	taskRunStatusMap := make(map[string]*v1alpha1.PipelineRunTaskRunStatus)
	taskRunStatusMap["faketaskrun"] = &v1alpha1.PipelineRunTaskRunStatus{
		Status: &v1alpha1.TaskRunStatus{
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Message: "Pending",
					},
				},
			}},
		},
	}

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
		Status: v1alpha1.PipelineRunStatus{
			TaskRuns: taskRunStatusMap,
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

	names, paNames, err := tl.GetTektonPipelinesWithActivePipelineActivity([]string{}, "fakecontext")
	assert.NoError(t, err)

	assert.Equal(t, 0, len(names))
	assert.Equal(t, 1, len(paNames))
}

func TestGetRunningBuildLogsNoBuildPods(t *testing.T) {
	jxClient, tektonClient, kubeClient, _, ns := getFakeClientsAndNs(t)
	tl := TektonLogger{
		JXClient:     jxClient,
		TektonClient: tektonClient,
		KubeClient:   kubeClient,
		Namespace:    ns,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}
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

	err := tl.GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1")
	assert.Error(t, err)
	assert.Equal(t, "the build pods for this build have been garbage collected and the log was not found in the long term storage bucket", err.Error())
}

func TestGetRunningBuildLogsWithPipelineRunButNoBuildPods(t *testing.T) {
	testCaseDir := path.Join("test_data")
	_, _, kubeClient, _, ns := getFakeClientsAndNs(t)

	pipelineRuns := tekton_helpers_test.AssertLoadPipelineRuns(t, testCaseDir)
	tektonClient := tektonMocks.NewSimpleClientset(pipelineRuns)
	structures := tekton_helpers_test.AssertLoadPipelineStructures(t, testCaseDir)
	jxClient := jxfake.NewSimpleClientset(structures)

	tl := TektonLogger{
		JXClient:     jxClient,
		TektonClient: tektonClient,
		Namespace:    ns,
		KubeClient:   kubeClient,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}

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

	err := tl.GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1")
	assert.Error(t, err)
	assert.Equal(t, "the build pods for this build have been garbage collected and the log was not found in the long term storage bucket", err.Error())
}

func TestGetRunningBuildLogsNoMatchingBuildPods(t *testing.T) {
	testCaseDir := path.Join("test_data")
	jxClient, tektonClient, _, _, ns := getFakeClientsAndNs(t)

	podsList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)
	kubeClient := kubeMocks.NewSimpleClientset(podsList)

	tl := TektonLogger{
		JXClient:     jxClient,
		TektonClient: tektonClient,
		KubeClient:   kubeClient,
		Namespace:    ns,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}

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

	err := tl.GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1")
	assert.Error(t, err)
	assert.Equal(t, "the build pods for this build have been garbage collected and the log was not found in the long term storage bucket", err.Error())
}

func TestGetRunningBuildLogsWithMatchingBuildPods(t *testing.T) {
	// https://github.com/jenkins-x/jx/issues/5171
	t.SkipNow()
	testCaseDir := path.Join("test_data")
	_, _, _, _, ns := getFakeClientsAndNs(t)

	podsList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)
	pipelineRuns := tekton_helpers_test.AssertLoadPipelineRuns(t, testCaseDir)
	kubeClient := kubeMocks.NewSimpleClientset(podsList)
	tektonClient := tektonMocks.NewSimpleClientset(pipelineRuns)
	structures := tekton_helpers_test.AssertLoadPipelineStructures(t, testCaseDir)
	jxClient := jxfake.NewSimpleClientset(structures)

	tl := TektonLogger{
		JXClient:     jxClient,
		TektonClient: tektonClient,
		KubeClient:   kubeClient,
		Namespace:    ns,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}

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

	bytesF, err := ioutil.ReadFile("/Users/daniel-gozalo/go/src/github.com/jenkins-x/jx/pkg/logs/test_data/multiple_stages/pipelinerun.yml")

	reader := bufio.NewReader(bytes.NewReader(bytesF))
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				fmt.Println("EOF")
				break
			}
		}
		fmt.Println(string(line))
	}

	err = tl.GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1")

	containers1, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[0])
	containers2, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[1])
	containersNumber := len(containers1) + len(containers2)

	assert.NoError(t, err)
	assert.Equal(t, containersNumber, len(tl.LogWriter.(*TestWriter).StreamLinesLogged))
}

func TestGetRunningBuildLogsForLegacyPipelineRunWithMatchingBuildPods(t *testing.T) {
	// https://github.com/jenkins-x/jx/issues/5171
	t.SkipNow()

	testCaseDir := path.Join("test_data", "legacy_pipeline_run")
	_, _, _, _, ns := getFakeClientsAndNs(t)

	podsList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)
	pipelineRuns := tekton_helpers_test.AssertLoadPipelineRuns(t, testCaseDir)
	kubeClient := kubeMocks.NewSimpleClientset(podsList)
	tektonClient := tektonMocks.NewSimpleClientset(pipelineRuns)
	structures := tekton_helpers_test.AssertLoadPipelineStructures(t, testCaseDir)
	jxClient := jxfake.NewSimpleClientset(structures)

	tl := TektonLogger{
		JXClient:     jxClient,
		TektonClient: tektonClient,
		KubeClient:   kubeClient,
		Namespace:    ns,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}

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

	err := tl.GetRunningBuildLogs(pa, "fakeowner/fakerepo/fakebranch/1")

	containers1, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[0])
	containers2, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[1])
	containersNumber := len(containers1) + len(containers2)

	assert.NoError(t, err)
	assert.Equal(t, containersNumber, len(tl.LogWriter.(*TestWriter).StreamLinesLogged))
}

func TestStreamPipelinePersistentLogsNotInBucket(t *testing.T) {
	_, _, _, commonOptions, _ := getFakeClientsAndNs(t)
	commonOptions.SkipAuthSecretsMerge = true

	tl := TektonLogger{
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
	}

	exampleLogLine := "This is an example log line"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		_, err := fmt.Fprintf(w, exampleLogLine)
		assert.NoError(t, err)
	}))

	logOutput := log.CaptureOutput(func() {
		err := tl.StreamPipelinePersistentLogs(server.URL, &commonOptions)
		assert.NoError(t, err)
	})

	assert.Contains(t, string(logOutput), "This is an example log line")
}

func TestStreamPipelinePersistentLogsInUnsupportedBucketProvider(t *testing.T) {
	_, _, _, commonOptions, _ := getFakeClientsAndNs(t)
	commonOptions.SkipAuthSecretsMerge = true
	tl := TektonLogger{
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
	}

	logOutput := log.CaptureOutput(func() {
		err := tl.StreamPipelinePersistentLogs("s3://nonSupportedBucket", &commonOptions)
		assert.NoError(t, err)
	})

	assert.Contains(t, string(logOutput), "The provided logsURL scheme is not supported: s3")
}

func TestGetRunningBuildLogsWithMultipleStages(t *testing.T) {
	// https://github.com/jenkins-x/jx/issues/5171
	t.SkipNow()
	testCaseDir := path.Join("test_data", "multiple_stages")
	_, _, _, _, ns := getFakeClientsAndNs(t)

	podsList := tekton_helpers_test.AssertLoadPods(t, testCaseDir)
	pipelineRuns := tekton_helpers_test.AssertLoadSinglePipelineRun(t, testCaseDir)
	kubeClient := kubeMocks.NewSimpleClientset(podsList)
	tektonClient := tektonMocks.NewSimpleClientset(pipelineRuns)
	structures := tekton_helpers_test.AssertLoadSinglePipelineStructure(t, testCaseDir)
	jxClient := jxfake.NewSimpleClientset(structures)

	tl := TektonLogger{
		KubeClient:   kubeClient,
		JXClient:     jxClient,
		TektonClient: tektonClient,
		Namespace:    ns,
		LogWriter: &TestWriter{
			StreamLinesLogged: make([]string, 0),
			SingleLinesLogged: make([]string, 0),
		},
		LogsRetrieverFunc: LogsProvider,
	}

	pa := &v1.PipelineActivity{
		ObjectMeta: v12.ObjectMeta{
			Name:      "abayer-js-test-repo-master-1",
			Namespace: ns,
			Labels: map[string]string{
				v1.LabelRepository: "js-test-repo",
				v1.LabelBranch:     "master",
				v1.LabelBuild:      "1",
				v1.LabelOwner:      "abayer",
			},
		},
		Spec: v1.PipelineActivitySpec{
			Build:         "1",
			GitBranch:     "master",
			GitRepository: "js-test-repo",
			GitOwner:      "abayer",
		},
	}

	logOutput := log.CaptureOutput(func() {
		err := tl.GetRunningBuildLogs(pa, "abayer/js-test-repo/master/1")
		assert.NoError(t, err)
	})

	aORb := regexp.MustCompile("Pod logs...")
	n := aORb.FindAllStringIndex(logOutput, -1)

	containers1, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[0])
	containers2, _, _ := kube.GetContainersWithStatusAndIsInit(&podsList.Items[1])
	containersNumber := len(containers1) + len(containers2)

	assert.Equal(t, containersNumber, len(n))
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

func (w *TestWriter) WriteLog(logLine LogLine) error {
	log.Logger().Info(logLine.Line)
	w.SingleLinesLogged = append(w.SingleLinesLogged, logLine.Line)
	return nil
}

func (w *TestWriter) StreamLog(lch <-chan LogLine, ech <-chan error) error {
	for {
		select {
		case l, ok := <-lch:
			if !ok {
				return nil
			}
			w.StreamLinesLogged = append(w.StreamLinesLogged, l.Line)
			log.Logger().Info(l.Line)
		case e := <-ech:
			fmt.Println(e)
			continue
		}
	}
}

func (w TestWriter) BytesLimit() int {
	return 0
}

func LogsProvider(pod *corev1.Pod, container *corev1.Container) (io.Reader, func(), error) {
	return bytes.NewReader([]byte("Pod logs...")), func() {
		//nothing to clean
	}, nil
}
