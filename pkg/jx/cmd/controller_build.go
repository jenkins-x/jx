package cmd

import (
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/collector"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/knative/build-pipeline/pkg/apis/pipeline"
	"github.com/pkg/errors"

	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	tektonclient "github.com/knative/build-pipeline/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/jenkins-x/jx/pkg/kube"
)

// ControllerBuildOptions are the flags for the commands
type ControllerBuildOptions struct {
	ControllerOptions

	Namespace          string
	InitGitCredentials bool

	EnvironmentCache *kube.EnvironmentNamespaceCache
}

// NewCmdControllerBuild creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdControllerBuild(commonOpts *CommonOptions) *cobra.Command {
	options := &ControllerBuildOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Runs the build controller",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		Aliases: []string{"builds"},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to watch or defaults to the current namespace")
	cmd.Flags().BoolVarP(&options.InitGitCredentials, "git-credentials", "", false, "If enable then lets run the 'jx step git credentials' step to initialise git credentials")
	return cmd
}

// Run implements this command
func (o *ControllerBuildOptions) Run() error {
	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}

	tektonClient, _, err := o.TektonClient()
	if err != nil {
		return err
	}

	tektonEnabled, err := kube.IsTektonEnabled(kubeClient, devNs)
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = devNs
	}

	o.EnvironmentCache = kube.CreateEnvironmentCache(jxClient, ns)

	if o.InitGitCredentials {
		err = o.setupGitCredentails()
		if err != nil {
			return err
		}
	}

	if tektonEnabled {
		pod := &corev1.Pod{}
		log.Infof("Watching for Pods in namespace %s\n", util.ColorInfo(ns))
		listWatch := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "pods", ns, fields.Everything())
		kube.SortListWatchByName(listWatch)
		_, controller := cache.NewInformer(
			listWatch,
			pod,
			time.Minute*10,
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					o.onPipelinePod(obj, kubeClient, jxClient, tektonClient, ns)
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					o.onPipelinePod(newObj, kubeClient, jxClient, tektonClient, ns)
				},
				DeleteFunc: func(obj interface{}) {
				},
			},
		)

		stop := make(chan struct{})
		go controller.Run(stop)
	} else {
		pod := &corev1.Pod{}
		log.Infof("Watching for Knative build pods in namespace %s\n", util.ColorInfo(ns))
		listWatch := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "pods", ns, fields.Everything())
		kube.SortListWatchByName(listWatch)
		_, controller := cache.NewInformer(
			listWatch,
			pod,
			time.Minute*10,
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					o.onPod(obj, kubeClient, jxClient, ns)
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					o.onPod(newObj, kubeClient, jxClient, ns)
				},
				DeleteFunc: func(obj interface{}) {
				},
			},
		)

		stop := make(chan struct{})
		go controller.Run(stop)
	}

	// Wait forever
	select {}
}

func (o *ControllerBuildOptions) onPod(obj interface{}, kubeClient kubernetes.Interface, jxClient versioned.Interface, ns string) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		log.Infof("Object is not a Pod %#v\n", obj)
		return
	}
	if pod != nil {
		o.handleStandalonePod(pod, kubeClient, jxClient, ns)
	}
}

func (o *ControllerBuildOptions) handleStandalonePod(pod *corev1.Pod, kubeClient kubernetes.Interface, jxClient versioned.Interface, ns string) {
	labels := pod.Labels
	if labels != nil {
		buildName := labels[builds.LabelBuildName]
		if buildName == "" {
			buildName = labels[builds.LabelOldBuildName]
		}
		if buildName == "" {
			buildName = labels[builds.LabelPipelineRunName]
		}
		if buildName != "" {
			if o.Verbose {
				log.Infof("Found build pod %s\n", pod.Name)
			}

			activities := jxClient.JenkinsV1().PipelineActivities(ns)
			key := o.createPromoteStepActivityKey(buildName, pod)
			if key != nil {
				name := ""
				err := util.Retry(time.Second*20, func() error {
					a, created, err := key.GetOrCreate(jxClient, ns)
					if err != nil {
						operation := "update"
						if created {
							operation = "create"
						}
						log.Warnf("Failed to %s PipelineActivities for build %s: %s\n", operation, buildName, err)
					}
					if o.updatePipelineActivity(kubeClient, ns, a, buildName, pod) {
						if o.Verbose {
							log.Infof("updating PipelineActivity %s\n", a.Name)
						}
						_, err := activities.Update(a)
						if err != nil {
							log.Warnf("Failed to update PipelineActivity %s due to: %s\n", a.Name, err.Error())
							name = a.Name
							return err
						}
					}
					return nil
				})
				if err != nil {
					log.Warnf("Failed to update PipelineActivities%s: %s\n", name, err)
				}
			}
		}
	}

}

func (o *ControllerBuildOptions) onPipelinePod(obj interface{}, kubeClient kubernetes.Interface, jxClient versioned.Interface, tektonClient tektonclient.Interface, ns string) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		log.Infof("Object is not a Pod %#v\n", obj)
		return
	}
	if pod != nil {
		if pod.Labels[pipeline.GroupName+pipeline.PipelineRunLabelKey] != "" {
			if pod.Labels[syntax.LabelStageName] != "" {
				prName := pod.Labels[pipeline.GroupName+pipeline.PipelineRunLabelKey]
				pri, err := tekton.CreatePipelineRunInfo(kubeClient, tektonClient, jxClient, ns, prName)
				if err != nil {
					log.Warnf("Error creating PipelineRunInfo for PipelineRun %s: %s\n", prName, err)
					return
				}
				if pri != nil {
					if o.Verbose {
						log.Infof("Found pipeline run %s\n", pri.Name)
					}

					activities := jxClient.JenkinsV1().PipelineActivities(ns)
					key := o.createPromoteStepActivityKeyFromRun(pri)
					if key != nil {
						name := ""
						err := util.Retry(time.Second*20, func() error {
							a, created, err := key.GetOrCreate(jxClient, ns)
							if err != nil {
								operation := "update"
								if created {
									operation = "create"
								}
								log.Warnf("Failed to %s PipelineActivities for build %s: %s\n", operation, pri.Name, err)
							}
							if o.updatePipelineActivityForRun(kubeClient, ns, a, pri) {
								if o.Verbose {
									log.Infof("updating PipelineActivity %s\n", a.Name)
								}
								_, err := activities.Update(a)
								if err != nil {
									log.Warnf("Failed to update PipelineActivity %s due to: %s\n", a.Name, err.Error())
									name = a.Name
									return err
								}
							}
							return nil
						})
						if err != nil {
							log.Warnf("Failed to update PipelineActivities%s: %s\n", name, err)
						}
					}
				}
			} else {
				o.handleStandalonePod(pod, kubeClient, jxClient, ns)
			}
		}
	}
}

// createPromoteStepActivityKey deduces the pipeline metadata from the Knative build pod
func (o *ControllerBuildOptions) createPromoteStepActivityKey(buildName string, pod *corev1.Pod) *kube.PromoteStepActivityKey {

	buildInfo := builds.CreateBuildPodInfo(pod)
	if buildInfo.GitURL == "" || buildInfo.GitInfo == nil {
		return nil
	}
	return &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:              buildInfo.Name,
			Pipeline:          buildInfo.Pipeline,
			Build:             buildInfo.Build,
			LastCommitSHA:     buildInfo.LastCommitSHA,
			LastCommitMessage: buildInfo.LastCommitMessage,
			LastCommitURL:     buildInfo.LastCommitURL,
			GitInfo:           buildInfo.GitInfo,
		},
	}
}

// createPromoteStepActivityKeyFromRun deduces the pipeline metadata from the pipeline run info
func (o *ControllerBuildOptions) createPromoteStepActivityKeyFromRun(pri *tekton.PipelineRunInfo) *kube.PromoteStepActivityKey {
	if pri.GitURL == "" || pri.GitInfo == nil {
		return nil
	}
	return &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:              pri.Name,
			Pipeline:          pri.Pipeline,
			Build:             pri.Build,
			LastCommitSHA:     pri.LastCommitSHA,
			LastCommitMessage: pri.LastCommitMessage,
			LastCommitURL:     pri.LastCommitURL,
			GitInfo:           pri.GitInfo,
		},
	}
}

func (o *ControllerBuildOptions) updatePipelineActivity(kubeClient kubernetes.Interface, ns string, activity *v1.PipelineActivity, buildName string, pod *corev1.Pod) bool {
	originYaml := toYamlString(activity)
	initContainersTerminated := len(pod.Status.InitContainerStatuses) > 0
	for _, c := range pod.Status.InitContainerStatuses {
		name := strings.Replace(strings.TrimPrefix(c.Name, "build-step-"), "-", " ", -1)
		title := strings.Title(name)
		_, stage, _ := kube.GetOrCreateStage(activity, title)

		running := c.State.Running
		terminated := c.State.Terminated

		var startedAt metav1.Time
		var finishedAt metav1.Time
		if terminated != nil {
			startedAt = terminated.StartedAt
			finishedAt = terminated.FinishedAt

			if !finishedAt.IsZero() {
				stage.CompletedTimestamp = &finishedAt
			}
		}
		if startedAt.IsZero() && running != nil {
			startedAt = running.StartedAt
		}

		if !startedAt.IsZero() {
			stage.StartedTimestamp = &startedAt
		}
		stage.Description = createStepDescription(c.Name, pod)

		if terminated != nil {
			if terminated.ExitCode == 0 {
				stage.Status = v1.ActivityStatusTypeSucceeded
			} else {
				stage.Status = v1.ActivityStatusTypeFailed
			}
		} else {
			if running != nil {
				stage.Status = v1.ActivityStatusTypeRunning
			} else {
				stage.Status = v1.ActivityStatusTypePending
			}
			initContainersTerminated = false
		}
	}
	spec := &activity.Spec
	var biggestFinishedAt metav1.Time

	allCompleted := true
	failed := false
	running := true
	for i := range spec.Steps {
		step := &spec.Steps[i]
		stage := step.Stage
		if stage != nil {
			stageFinished := spec.Status.IsTerminated()
			if stage.StartedTimestamp != nil && spec.StartedTimestamp == nil {
				spec.StartedTimestamp = stage.StartedTimestamp
			}
			if stage.CompletedTimestamp != nil {
				t := stage.CompletedTimestamp
				if !t.IsZero() {
					stageFinished = true
					if biggestFinishedAt.IsZero() || t.After(biggestFinishedAt.Time) {
						biggestFinishedAt = *t
					}
				}
			}
			if stageFinished {
				if stage.Status != v1.ActivityStatusTypeSucceeded {
					failed = true
				}
			} else {
				allCompleted = false
			}
			if stage.Status == v1.ActivityStatusTypeRunning {
				running = true
			}
			if stage.Status == v1.ActivityStatusTypeRunning || stage.Status == v1.ActivityStatusTypePending {
				allCompleted = false
			}
		}
	}

	if !allCompleted && initContainersTerminated {
		allCompleted = true
	}
	if allCompleted {
		if failed {
			spec.Status = v1.ActivityStatusTypeFailed
		} else {
			spec.Status = v1.ActivityStatusTypeSucceeded
		}
		if !biggestFinishedAt.IsZero() {
			spec.CompletedTimestamp = &biggestFinishedAt
		}
		// lets ensure we overwrite any canonical jenkins build URL thats generated automatically
		if spec.BuildLogsURL == "" || !strings.Contains(spec.BuildLogsURL, pod.Name) {
			podInterface := kubeClient.CoreV1().Pods(ns)

			envName := kube.LabelValueDevEnvironment
			devEnv := o.EnvironmentCache.Item(envName)
			var location *v1.StorageLocation
			settings := &devEnv.Spec.TeamSettings
			if devEnv == nil {
				log.Warnf("No Environment %s found\n", envName)
			} else {
				location = settings.StorageLocationOrDefault(kube.ClassificationLogs)
			}
			if location == nil {
				location = &v1.StorageLocation{}
			}
			if location.IsEmpty() {
				location.GitURL = activity.Spec.GitURL
				if location.GitURL == "" {
					log.Warnf("No GitURL on PipelineActivity %s\n", activity.Name)
				}
			}
			logURL, err := o.generateBuildLogURL(podInterface, ns, activity, buildName, pod, location, settings, o.InitGitCredentials)
			if err != nil {
				if o.Verbose {
					log.Warnf("%s\n", err)
				}
			}
			if logURL != "" {
				spec.BuildLogsURL = logURL
			}
		}
	} else {
		if running {
			spec.Status = v1.ActivityStatusTypeRunning
		} else {
			spec.Status = v1.ActivityStatusTypePending
		}
	}

	// lets compare YAML in case we modify arrays in place on a copy (such as the steps) and don't detect we changed things
	newYaml := toYamlString(activity)
	return originYaml != newYaml
}

func (o *ControllerBuildOptions) updatePipelineActivityForRun(kubeClient kubernetes.Interface, ns string, activity *v1.PipelineActivity, pri *tekton.PipelineRunInfo) bool {
	originYaml := toYamlString(activity)
	for _, stage := range pri.Stages {
		updateForStage(stage, activity)
	}

	spec := &activity.Spec
	var biggestFinishedAt metav1.Time

	allCompleted := true
	failed := false
	running := true
	for i := range spec.Steps {
		step := &spec.Steps[i]
		stage := step.Stage
		if stage != nil {
			stageFinished := spec.Status.IsTerminated()
			if stage.StartedTimestamp != nil && spec.StartedTimestamp == nil {
				spec.StartedTimestamp = stage.StartedTimestamp
			}
			if stage.CompletedTimestamp != nil {
				t := stage.CompletedTimestamp
				if !t.IsZero() {
					stageFinished = true
					if biggestFinishedAt.IsZero() || t.After(biggestFinishedAt.Time) {
						biggestFinishedAt = *t
					}
				}
			}
			if stageFinished {
				if stage.Status != v1.ActivityStatusTypeSucceeded {
					failed = true
				}
			} else {
				allCompleted = false
			}
			if stage.Status == v1.ActivityStatusTypeRunning {
				running = true
			}
			if stage.Status == v1.ActivityStatusTypeRunning || stage.Status == v1.ActivityStatusTypePending {
				allCompleted = false
			}
		}
	}

	if allCompleted {
		if failed {
			spec.Status = v1.ActivityStatusTypeFailed
		} else {
			spec.Status = v1.ActivityStatusTypeSucceeded
		}
		if !biggestFinishedAt.IsZero() {
			spec.CompletedTimestamp = &biggestFinishedAt
		}
		// TODO: Not yet sure how to handle BuildLogsURL
	} else {
		if running {
			spec.Status = v1.ActivityStatusTypeRunning
		} else {
			spec.Status = v1.ActivityStatusTypePending
		}
	}

	// lets compare YAML in case we modify arrays in place on a copy (such as the steps) and don't detect we changed things
	newYaml := toYamlString(activity)
	return originYaml != newYaml
}

func updateForStage(si *tekton.StageInfo, a *v1.PipelineActivity) {
	_, stage, _ := kube.GetOrCreateStage(a, si.GetStageNameIncludingParents())
	initContainersTerminated := false

	if si.Pod != nil {
		pod := si.Pod
		initContainersTerminated = len(pod.Status.InitContainerStatuses) > 0
		for _, c := range pod.Status.InitContainerStatuses {
			name := strings.Replace(strings.TrimPrefix(c.Name, "build-step-"), "-", " ", -1)
			title := strings.Title(name)
			step, _ := kube.GetOrCreateStepInStage(stage, title)
			running := c.State.Running
			terminated := c.State.Terminated

			var startedAt metav1.Time
			var finishedAt metav1.Time
			if terminated != nil {
				startedAt = terminated.StartedAt
				finishedAt = terminated.FinishedAt

				if !finishedAt.IsZero() {
					step.CompletedTimestamp = &finishedAt
				}
			}
			if startedAt.IsZero() && running != nil {
				startedAt = running.StartedAt
			}

			if !startedAt.IsZero() {
				step.StartedTimestamp = &startedAt
			}
			step.Description = createStepDescription(c.Name, pod)

			if terminated != nil {
				if terminated.ExitCode == 0 {
					step.Status = v1.ActivityStatusTypeSucceeded
				} else {
					step.Status = v1.ActivityStatusTypeFailed
				}
			} else {
				if running != nil {
					step.Status = v1.ActivityStatusTypeRunning
				} else {
					step.Status = v1.ActivityStatusTypePending
				}
				initContainersTerminated = false
			}
		}
	}

	for _, nested := range si.Parallel {
		updateForStage(nested, a)
	}

	for _, nested := range si.Stages {
		updateForStage(nested, a)
	}

	var biggestFinishedAt metav1.Time

	childStageNames := si.GetFullChildStageNames(false)

	if len(childStageNames) > 0 {
		var childStages []*v1.StageActivityStep
		for _, s := range a.Spec.Steps {
			if s.Stage != nil {
				for _, c := range childStageNames {
					if s.Stage.Name == c {
						childStages = append(childStages, s.Stage)
					}
				}
			}
		}

		childrenCompleted := true
		childrenFailed := false
		childrenRunning := true

		for _, child := range childStages {
			childFinished := child.Status.IsTerminated()
			if child.StartedTimestamp != nil && stage.StartedTimestamp == nil {
				stage.StartedTimestamp = child.StartedTimestamp
			}
			if child.CompletedTimestamp != nil {
				t := child.CompletedTimestamp
				if !t.IsZero() {
					childFinished = true
					if biggestFinishedAt.IsZero() || t.After(biggestFinishedAt.Time) {
						biggestFinishedAt = *t
					}
				}
			}
			if childFinished {
				if child.Status != v1.ActivityStatusTypeSucceeded {
					childrenFailed = true
				}
			} else {
				childrenCompleted = false
			}
			if child.Status == v1.ActivityStatusTypeRunning {
				childrenRunning = true
			}
			if child.Status == v1.ActivityStatusTypeRunning || child.Status == v1.ActivityStatusTypePending {
				childrenCompleted = false
			}
		}

		if childrenCompleted {
			if childrenFailed {
				stage.Status = v1.ActivityStatusTypeFailed
			} else {
				stage.Status = v1.ActivityStatusTypeSucceeded
			}
			if !biggestFinishedAt.IsZero() {
				stage.CompletedTimestamp = &biggestFinishedAt
			}
		} else {
			if childrenRunning {
				stage.Status = v1.ActivityStatusTypeRunning
			} else {
				stage.Status = v1.ActivityStatusTypePending
			}
		}
	} else {
		allCompleted := false
		failed := false
		running := si.Pod != nil
		for i := range stage.Steps {
			step := &stage.Steps[i]
			if stage != nil {
				stepFinished := step.Status.IsTerminated()
				if step.StartedTimestamp != nil && stage.StartedTimestamp == nil {
					stage.StartedTimestamp = step.StartedTimestamp
				}
				if step.CompletedTimestamp != nil {
					t := step.CompletedTimestamp
					if !t.IsZero() {
						stepFinished = true
						if biggestFinishedAt.IsZero() || t.After(biggestFinishedAt.Time) {
							biggestFinishedAt = *t
						}
					}
				}
				if stepFinished {
					if step.Status != v1.ActivityStatusTypeSucceeded {
						failed = true
					}
				} else {
					allCompleted = false
				}
				if step.Status == v1.ActivityStatusTypeRunning {
					running = true
				}
				if step.Status == v1.ActivityStatusTypeRunning || step.Status == v1.ActivityStatusTypePending {
					allCompleted = false
				}
			}
		}

		if !allCompleted && initContainersTerminated {
			allCompleted = true
		}
		if allCompleted {
			if failed {
				stage.Status = v1.ActivityStatusTypeFailed
			} else {
				stage.Status = v1.ActivityStatusTypeSucceeded
			}
			if !biggestFinishedAt.IsZero() {
				stage.CompletedTimestamp = &biggestFinishedAt
			}
		} else {
			if running {
				stage.Status = v1.ActivityStatusTypeRunning
			} else {
				stage.Status = v1.ActivityStatusTypePending
			}
		}
	}
}

// toYamlString returns the YAML string or error when marshalling the given resource
func toYamlString(resource interface{}) string {
	data, err := yaml.Marshal(resource)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// generates the build log URL and returns the URL
func (o *CommonOptions) generateBuildLogURL(podInterface typedcorev1.PodInterface, ns string, activity *v1.PipelineActivity, buildName string, pod *corev1.Pod, location *v1.StorageLocation, settings *v1.TeamSettings, initGitCredentials bool) (string, error) {

	coll, err := collector.NewCollector(location, settings, o.Git())
	if err != nil {
		return "", errors.Wrapf(err, "could not create Collector for pod %s in namespace %s with settings %#v", pod.Name, ns, settings)
	}

	data, err := builds.GetBuildLogsForPod(podInterface, pod)
	if err != nil {
		// probably due to not being available yet
		return "", errors.Wrapf(err, "failed to get build log for pod %s in namespace %s", pod.Name, ns)
	}

	if o.Verbose {
		log.Infof("got build log for pod: %s PipelineActivity: %s with bytes: %d\n", pod.Name, activity.Name, len(data))
	}

	if initGitCredentials {
		gc := &StepGitCredentialsOptions{}
		gc.CommonOptions = o
		gc.BatchMode = true
		log.Info("running: jx step git credentials\n")
		err = gc.Run()
		if err != nil {
			return "", errors.Wrapf(err, "Failed to setup git credentials")
		}
	}

	owner := activity.Spec.GitOwner
	repository := activity.RepositoryName()
	branch := activity.BranchName()
	buildNumber := activity.Spec.Build
	if buildNumber == "" {
		buildNumber = "1"
	}

	pathDir := filepath.Join("jenkins-x", "logs", owner, repository, branch)
	fileName := filepath.Join(pathDir, buildNumber+".log")

	url, err := coll.CollectData(data, fileName)
	if err != nil {
		return url, errors.Wrapf(err, "failed to collect build log for pod %s in namespace %s", pod.Name, ns)
	}
	return url, nil
}

// createStepDescription uses the spec of the init container to return a description
func createStepDescription(initContainerName string, pod *corev1.Pod) string {
	for _, c := range pod.Spec.InitContainers {
		args := c.Args
		if c.Name == initContainerName && len(args) > 0 {
			if args[0] == "-url" && len(args) > 1 {
				return args[1]
			}
		}
	}
	return ""
}

// DigitSuffix outputs digital suffix
func DigitSuffix(text string) string {
	answer := ""
	for {
		l := len(text)
		if l == 0 {
			return answer
		}
		lastChar := text[l-1:]
		for _, rune := range lastChar {
			if !unicode.IsDigit(rune) {
				return answer
			}
			break
		}
		answer = lastChar + answer
		text = text[0 : l-1]
	}
}
