package logs

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/jenkins-x/jx/v2/pkg/auth"
	"github.com/jenkins-x/jx/v2/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/v2/pkg/config"
	"github.com/jenkins-x/jx/v2/pkg/errorutil"
	"github.com/jenkins-x/jx/v2/pkg/kube/naming"

	"github.com/fatih/color"
	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/builds"
	"github.com/jenkins-x/jx/v2/pkg/cloud/factory"
	"github.com/jenkins-x/jx/v2/pkg/cloud/gke"
	"github.com/jenkins-x/jx/v2/pkg/cmd/step"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/tekton"
	"github.com/pkg/errors"
	tektonapis "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	knativeapis "knative.dev/pkg/apis"
)

// TektonLogger contains the necessary clients and the namespace to get data from the cluster, an implementation of
// LogWriter to write logs to and a logs retriever function to override the default way to obtain logs
type TektonLogger struct {
	JXClient          versioned.Interface
	TektonClient      tektonclient.Interface
	KubeClient        kubernetes.Interface
	Namespace         string
	BytesLimit        int64
	FailIfPodFails    bool
	LogsRetrieverFunc retrieverFunc
	err               error
}

// Err returns the last error that occurred during streaming logs.
// It should be checked after the log stream channel has been closed.
func (t *TektonLogger) Err() error {
	return t.err
}

// retrieverFunc is a func signature used to define the LogsRetrieverFunc in TektonLogger
type retrieverFunc func(pod *corev1.Pod, container *corev1.Container, limitBytes int64, c kubernetes.Interface) (io.ReadCloser, error)

// LogLine is the object sent to and received from the channels in the StreamLog and WriteLog functions
// defined by LogWriter
type LogLine struct {
	Line       string
	ShouldMask bool
}

// GetTektonPipelinesWithActivePipelineActivity returns list of all PipelineActivities with corresponding Tekton PipelineRuns ordered by the PipelineRun creation timestamp and a map to obtain its reference once a name has been selected
func (t TektonLogger) GetTektonPipelinesWithActivePipelineActivity(filters []string) ([]string, map[string]*v1.PipelineActivity, error) {
	labelsFilter := strings.Join(filters, ",")
	paList, err := t.JXClient.JenkinsV1().PipelineActivities(t.Namespace).List(metav1.ListOptions{
		LabelSelector: labelsFilter,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "there was a problem getting the PipelineActivities")
	}

	sort.Slice(paList.Items, func(i, j int) bool {
		return paList.Items[i].CreationTimestamp.After(paList.Items[j].CreationTimestamp.Time)
	})

	paMap := make(map[string]*v1.PipelineActivity)
	for _, pa := range paList.Items {
		p := pa
		paMap[createPipelineActivityName(p.Labels, p.Spec.Build)] = &p
	}

	tektonPRs, _ := t.TektonClient.TektonV1alpha1().PipelineRuns(t.Namespace).List(metav1.ListOptions{
		LabelSelector: labelsFilter,
	})

	// Handle the old "repo" tag as well for legacy purposes.
	if len(tektonPRs.Items) == 0 {
		labelsFilter = strings.Replace(labelsFilter, "repository=", "repo=", 1)
		tektonPRs, _ = t.TektonClient.TektonV1alpha1().PipelineRuns(t.Namespace).List(metav1.ListOptions{
			LabelSelector: labelsFilter,
		})
	}

	prMap := make(map[string][]*tektonapis.PipelineRun)
	for _, pr := range tektonPRs.Items {
		p := pr
		prStatus := p.Status.GetCondition(knativeapis.ConditionSucceeded)
		// Don't include any pipeline runs that failed due to Tekton race conditions and were auto-restarted by Prow
		if prStatus != nil && prStatus.IsFalse() &&
			(strings.Contains(prStatus.Message, "can't be found:pipeline.tekton.dev") ||
				strings.Contains(prStatus.Message, "it contains Tasks that don't exist")) {
			continue
		}
		prBuildNumber := p.Labels[v1.LabelBuild]
		if prBuildNumber == "" {
			prBuildNumber = findLegacyPipelineRunBuildNumber(&p)
		}
		paName := createPipelineActivityName(p.Labels, prBuildNumber)
		if _, exists := prMap[paName]; !exists {
			prMap[paName] = []*tektonapis.PipelineRun{}
		}
		prMap[paName] = append(prMap[paName], &p)
	}

	var names []string
	for _, pa := range paList.Items {
		paName := createPipelineActivityName(pa.Labels, pa.Spec.Build)
		if _, exists := prMap[paName]; exists {
			hasNonPendingPR := false
			for _, pr := range prMap[paName] {
				if tekton.PipelineRunIsNotPending(pr) {
					hasNonPendingPR = true
				}
			}
			if hasNonPendingPR {
				names = append(names, paName)
			}
		} else if pa.Spec.CompletedTimestamp != nil {
			names = append(names, paName)
		}
	}

	return names, paMap, nil
}

func createPipelineActivityName(labels map[string]string, buildNumber string) string {
	repository := labels[v1.LabelRepository]
	// The label is called "repo" in the PipelineRun CRD and "repository" in the PipelineActivity CRD
	if repository == "" {
		repository = labels["repo"]
	}
	baseName := fmt.Sprintf("%s/%s/%s #%s",
		naming.ToValidName(labels[v1.LabelOwner]),
		naming.ToValidName(repository),
		naming.ToValidName(labels[v1.LabelBranch]),
		strings.ToLower(buildNumber))

	context := labels[v1.LabelContext]
	if context != "" {
		return fmt.Sprintf("%s %s", baseName, naming.ToValidName(context))
	}

	return baseName
}

func findLegacyPipelineRunBuildNumber(pipelineRun *tektonapis.PipelineRun) string {
	var buildNumber string
	for _, p := range pipelineRun.Spec.Params {
		if p.Name == "build_id" && p.Value.Type == tektonapis.ParamTypeString {
			buildNumber = p.Value.StringVal
		}
	}
	return buildNumber
}

func getPipelineRunsForActivity(pa *v1.PipelineActivity, tektonClient tektonclient.Interface) (map[string]tektonapis.PipelineRun, error) {
	filters := []string{
		fmt.Sprintf("%s=%s", v1.LabelOwner, pa.Spec.GitOwner),
		fmt.Sprintf("%s=%s", v1.LabelRepository, pa.Spec.GitRepository),
		fmt.Sprintf("%s=%s", v1.LabelBranch, pa.Spec.GitBranch),
	}

	tektonPRs, err := tektonClient.TektonV1alpha1().PipelineRuns(pa.Namespace).List(metav1.ListOptions{
		LabelSelector: strings.Join(filters, ","),
	})
	if err != nil {
		return nil, err
	}
	// For legacy purposes, look for the old "repo" label as well.
	if len(tektonPRs.Items) == 0 {
		tektonPRs, err = tektonClient.TektonV1alpha1().PipelineRuns(pa.Namespace).List(metav1.ListOptions{
			LabelSelector: strings.Replace(strings.Join(filters, ","), "repository=", "repo=", 1),
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(tektonPRs.Items, func(i, j int) bool {
		return tektonPRs.Items[i].CreationTimestamp.Before(&tektonPRs.Items[j].CreationTimestamp)
	})

	runs := make(map[string]tektonapis.PipelineRun)
	for _, pr := range tektonPRs.Items {
		pipelineRun := pr
		buildNumber := pipelineRun.Labels[tekton.LabelBuild]
		if buildNumber == "" {
			buildNumber = findLegacyPipelineRunBuildNumber(&pipelineRun)
		}
		pipelineType := pipelineRun.Labels[tekton.LabelType]
		if pipelineType == "" {
			pipelineType = tekton.BuildPipeline.String()
		}
		if buildNumber == pa.Spec.Build {
			runs[pipelineType] = pipelineRun
		}
	}

	return runs, nil
}

// GetRunningBuildLogs obtains the logs of the provided PipelineActivity and streams the running build pods' logs using the provided LogWriter
func (t *TektonLogger) GetRunningBuildLogs(pa *v1.PipelineActivity, buildName string, noWaitForRuns bool) <-chan LogLine {
	ch := make(chan LogLine)
	go func() {
		defer close(ch)
		err := t.getRunningBuildLogs(pa, buildName, noWaitForRuns, ch)
		if err != nil {
			t.err = err
		}
	}()
	return ch
}

func (t TektonLogger) getRunningBuildLogs(pa *v1.PipelineActivity, buildName string, noWaitForRuns bool, out chan<- LogLine) error {
	loggedAllRunsForActivity := false
	foundLogs := false

	// Make sure we check again for the build pipeline if we just get the metapipeline initially, assuming the metapipeline succeeds
	for !loggedAllRunsForActivity {
		runsByType, err := getPipelineRunsForActivity(pa, t.TektonClient)
		if err != nil {
			return errors.Wrapf(err, "failed to get PipelineRun names for activity %s in namespace %s", pa.Name, pa.Namespace)
		}

		var runToLog *tektonapis.PipelineRun
		metaPr, hasMetaPr := runsByType[tekton.MetaPipeline.String()]
		buildPr, hasBuildPr := runsByType[tekton.BuildPipeline.String()]
		var metaStatus *knativeapis.Condition
		if hasMetaPr {
			metaStatus = metaPr.Status.GetCondition(knativeapis.ConditionSucceeded)
		}
		// If we don't have a build run yet, the build run doesn't yet have pods, or none of the build run pods are out
		// of pending yet, get the logs for the metapipeline.
		if hasMetaPr && (!hasBuildPr || !tekton.PipelineRunIsNotPending(&buildPr)) {
			runToLog = &metaPr

			// If we have a metapipeline and the metapipeline is failed, the pipeline as a whole is complete
			if metaStatus != nil && metaStatus.Status == corev1.ConditionFalse {
				loggedAllRunsForActivity = true
			}
		} else if hasBuildPr {
			// Log the build pipeline
			runToLog = &buildPr
			loggedAllRunsForActivity = true
		}

		// Assuming we have a run to log, go get its logs, looping until we've seen all stages for that run.
		if runToLog != nil {
			structure, err := tekton.StructureForPipelineRun(t.JXClient, pa.Namespace, runToLog)
			if err != nil {
				return errors.Wrapf(err, "failed to get pipeline structure for %s in namespace %s", runToLog.Name, pa.Namespace)
			}

			allStages := structure.GetAllStagesWithSteps()
			var stagesToCheckCount int
			// If the pipeline run is done, we only care about logs from the pods it actually ran.
			if runToLog.IsDone() || pa.Spec.Status.IsTerminated() {
				// Add all stages that actually ran while ignoring ones that were never executed, since the run is done.
				stagesToCheckCount = len(runToLog.Status.TaskRuns)
			} else {
				stagesToCheckCount = len(allStages)
			}
			stagesSeen := make(map[string]bool)

			// Repeat until we've seen pods for all stages
			for stagesToCheckCount > len(stagesSeen) {
				pods, err := builds.GetPipelineRunPods(t.KubeClient, pa.Namespace, runToLog.Name)
				if err != nil {
					return errors.Wrapf(err, "failed to get pods for pipeline run %s in namespace %s", runToLog.Name, pa.Namespace)
				}

				sort.Slice(pods, func(i, j int) bool {
					return pods[i].CreationTimestamp.Before(&pods[j].CreationTimestamp)
				})

				for _, pod := range pods {
					stageName := pod.Labels["jenkins.io/task-stage-name"]
					params := builds.CreateBuildPodInfo(pod)
					if _, seen := stagesSeen[stageName]; !seen && params.Organisation == pa.Spec.GitOwner && params.Repository == pa.Spec.GitRepository &&
						strings.ToLower(params.Branch) == strings.ToLower(pa.Spec.GitBranch) && params.Build == pa.Spec.Build {
						stagesSeen[stageName] = true
						foundLogs = true
						err := t.getContainerLogsFromPod(pod, pa, buildName, stageName, out)
						if err != nil {
							return errors.Wrapf(err, "failed to obtain the logs for build %s and stage %s", buildName, stageName)
						}
					}
				}
				if !foundLogs {
					break
				}
			}
		}
		if !foundLogs {
			break
		}

		// Flag used for testing - don't loop forever waiting for the build run if it's pending
		if noWaitForRuns && !tekton.PipelineRunIsNotPending(&buildPr) {
			loggedAllRunsForActivity = true
		}
	}
	if !foundLogs {
		return errors.New("the build pods for this build have been garbage collected and the log was not found in the long term storage bucket")
	}

	return nil
}

func (t *TektonLogger) getContainerLogsFromPod(pod *corev1.Pod, pa *v1.PipelineActivity, buildName string, stageName string, out chan<- LogLine) error {
	infoColor := color.New(color.FgGreen)
	infoColor.EnableColor()
	errorColor := color.New(color.FgRed)
	errorColor.EnableColor()
	containers, _, _ := kube.GetContainersWithStatusAndIsInit(pod)
	for i, initContainer := range containers {
		ic := initContainer
		pod, err := t.waitForContainerToStart(pa.Namespace, pod, i, stageName, out)
		out <- LogLine{
			Line: fmt.Sprintf("\nShowing logs for build %v stage %s and container %s",
				infoColor.Sprintf(buildName), infoColor.Sprintf(stageName), infoColor.Sprintf(ic.Name)),
		}
		if err != nil {
			return errors.Wrapf(err, "there was a problem writing a single line into the logs writer")
		}
		err = t.fetchLogsToChannel(pa.Namespace, pod, &ic, out)
		if err != nil {
			return errors.Wrap(err, "couldn't fetch logs into the logs channel")
		}
		if hasStepFailed(pod, i, t.KubeClient, pa.Namespace) {
			out <- LogLine{
				Line: errorColor.Sprintf("\nPipeline failed on stage '%s' : container '%s'. The execution of the pipeline has stopped.", stageName, ic.Name),
			}
			if err != nil {
				return err
			}
			if t.FailIfPodFails {
				return errors.Errorf("Pipeline failed on stage '%s' : container '%s'. The execution of the pipeline has stopped.", stageName, ic.Name)
			}
			break
		}
	}
	return nil
}

func (t *TektonLogger) fetchLogsToChannel(ns string, pod *corev1.Pod, container *corev1.Container, out chan<- LogLine) error {
	logsRetrieverFunc := t.LogsRetrieverFunc
	if logsRetrieverFunc == nil {
		logsRetrieverFunc = retrieveLogsFromPod
	}
	reader, err := logsRetrieverFunc(pod, container, t.BytesLimit, t.KubeClient)
	if err != nil {
		return err
	}
	defer reader.Close()
	return writeStreamLines(reader, out)
}

func writeStreamLines(reader io.Reader, out chan<- LogLine) error {
	buffReader := bufio.NewReader(reader)
	for {
		line, _, err := buffReader.ReadLine()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Wrap(err, "failed to read stream")
		}
		out <- LogLine{Line: string(line), ShouldMask: true}
	}
}

func hasStepFailed(pod *corev1.Pod, stepNumber int, kubeClient kubernetes.Interface, ns string) bool {
	pod, err := kubeClient.CoreV1().Pods(ns).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		log.Logger().Error("couldn't find the updated pod to check the step status")
		return false
	}
	_, containerStatus, _ := kube.GetContainersWithStatusAndIsInit(pod)
	if containerStatus[stepNumber].State.Terminated != nil && containerStatus[stepNumber].State.Terminated.ExitCode != 0 {
		return true
	}
	return false
}

func (t TektonLogger) waitForContainerToStart(ns string, pod *corev1.Pod, idx int, stageName string, out chan<- LogLine) (*corev1.Pod, error) {
	if pod.Status.Phase == corev1.PodFailed {
		return pod, nil
	}
	if kube.HasContainerStarted(pod, idx) {
		return pod, nil
	}
	containerName := ""
	containers, _, _ := kube.GetContainersWithStatusAndIsInit(pod)
	if idx < len(containers) {
		containerName = containers[idx].Name
	}
	// This method will be executed by both the CLI and the UI, we don't know if the UI has color enabled, so we are using a local instance instead of the global one
	c := color.New(color.FgGreen)
	c.EnableColor()
	out <- LogLine{
		Line: fmt.Sprintf("\nwaiting for stage %s : container %s to start...\n", c.Sprintf(stageName), c.Sprintf(containerName)),
	}
	for {
		time.Sleep(time.Second)
		p, err := t.KubeClient.CoreV1().Pods(ns).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return p, errors.Wrapf(err, "failed to load pod %s", pod.Name)
		}
		if kube.HasContainerStarted(p, idx) {
			return p, nil
		}
	}
}

// StreamPipelinePersistentLogs reads logs from the provided bucket URL and writes them using the provided LogWriter
func (t *TektonLogger) StreamPipelinePersistentLogs(logsURL string, authSvc auth.ConfigService) <-chan LogLine {
	ch := make(chan LogLine)
	go func() {
		defer close(ch)
		err := t.streamPipelinePersistentLogs(logsURL, authSvc, ch)
		if err != nil {
			t.err = err
		}
	}()
	return ch
}

func (t *TektonLogger) streamPipelinePersistentLogs(logsURL string, authSvc auth.ConfigService, out chan<- LogLine) error {
	u, err := url.Parse(logsURL)
	if err != nil {
		return errors.Wrapf(err, "unable to parse logs URL %s to retrieve scheme", logsURL)
	}
	switch u.Scheme {
	case "gs":
		reader, err := performProviderDownload(logsURL, t.JXClient, t.Namespace)
		if err != nil {
			// TODO: This is only here as long as we keep supporting non boot clusters, as GKE are the only ones with LTS supported outside of boot
			reader, err2 := gke.StreamTransferFileFromBucket(logsURL)
			if err2 != nil {
				return errorutil.CombineErrors(err, err2)
			}
			return t.streamPipedLogs(reader, out)
		}
		return t.streamPipedLogs(reader, out)
	case "s3":
		reader, err := performProviderDownload(logsURL, t.JXClient, t.Namespace)
		if err != nil {
			return errors.Wrap(err, "there was a problem downloading logs from s3 bucket")
		}
		return t.streamPipedLogs(reader, out)
	case "http", "https":
		reader, err := downloadLogFile(logsURL, authSvc)
		if err != nil {
			return errors.Wrapf(err, "there was a problem obtaining the log file from the github pages URL %s", logsURL)
		}
		return t.streamPipedLogs(reader, out)
	default:
		out <- LogLine{
			Line: fmt.Sprintf("The provided logsURL scheme is not supported: %s", u.Scheme),
		}
	}
	return nil
}

func (t *TektonLogger) streamPipedLogs(src io.ReadCloser, out chan<- LogLine) (err error) {
	defer func() {
		if e := src.Close(); e != nil && err == nil {
			err = e
		}
	}()
	scanner := bufio.NewScanner(src)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		text := scanner.Text()
		out <- LogLine{Line: text}
		if t.FailIfPodFails && strings.Contains(text, "The execution of the pipeline has stopped.") {
			return errors.New("the execution of the pipeline has stopped")
		}
	}
	return nil
}

// Uses the same signature as retrieverFunc so it can be used in TektonLogger
func retrieveLogsFromPod(pod *corev1.Pod, container *corev1.Container, limitBytes int64, client kubernetes.Interface) (io.ReadCloser, error) {
	options := &corev1.PodLogOptions{
		Container: container.Name,
		Follow:    true,
	}
	if limitBytes > 0 {
		options.LimitBytes = &limitBytes
	}
	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, options)
	stream, err := req.Stream()
	if err != nil {
		return nil, errors.Wrapf(err, "there was an error creating the logs stream for pod %s", pod.Name)
	}
	return stream, nil
}

func downloadLogFile(logsURL string, authSvc auth.ConfigService) (io.ReadCloser, error) {
	reader, err := buckets.ReadURL(logsURL, 30*time.Second, step.CreateBucketHTTPFn(authSvc))
	return reader, err
}

func performProviderDownload(logsURL string, jxClient versioned.Interface, ns string) (io.ReadCloser, error) {
	provider, err := NewBucketProviderFromTeamSettingsConfiguration(jxClient, ns)
	if err != nil {
		return nil, errors.Wrap(err, "There was a problem obtaining a Bucket provider for bucket scheme gs://")
	}
	return provider.DownloadFileFromBucket(logsURL)
}

func NewBucketProviderFromTeamSettingsConfiguration(jxClient versioned.Interface, ns string) (buckets.Provider, error) {
	teamSettings, err := kube.GetDevEnvTeamSettings(jxClient, ns)
	if err != nil {
		return nil, errors.Wrap(err, "error obtaining the dev environment teamSettings to select the correct bucket provider")
	}
	requirements, err := config.GetRequirementsConfigFromTeamSettings(teamSettings)
	if err != nil || requirements == nil {
		return nil, errorutil.CombineErrors(err, errors.New("error obtaining the requirements file to decide bucket provider"))
	}
	return factory.NewBucketProvider(requirements), nil
}
