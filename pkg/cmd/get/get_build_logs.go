package get

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/logs"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/knative/pkg/apis"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetBuildLogsOptions the command line options
type GetBuildLogsOptions struct {
	GetOptions

	Tail                    bool
	Wait                    bool
	BuildFilter             builds.BuildPodInfoFilter
	JenkinsSelector         opts.JenkinsSelectorOptions
	CurrentFolder           bool
	WaitForPipelineDuration time.Duration
}

// CLILogWriter is an implementation of logs.LogWriter that will show logs in the standard output
type CLILogWriter struct {
	*opts.CommonOptions
}

var (
	get_build_log_long = templates.LongDesc(`
		Display a build log

`)

	get_build_log_example = templates.Examples(`
		# Display a build log - with the user choosing which repo + build to view
		jx get build log

		# Pick a build to view the log based on the repo cheese
		jx get build log --repo cheese

		# Pick a pending knative build to view the log based 
		jx get build log -p

		# Pick a pending knative build to view the log based on the repo cheese
		jx get build log --repo cheese -p

		# Pick a knative build for the 1234 Pull Request on the repo cheese
		jx get build log --repo cheese --branch PR-1234

		# View the build logs for a specific tekton build pod
		jx get build log --pod my-pod-name
	`)
)

// NewCmdGetBuildLogs creates the command
func NewCmdGetBuildLogs(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetBuildLogsOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "log [flags]",
		Short:   "Display a build log",
		Long:    get_build_log_long,
		Example: get_build_log_example,
		Aliases: []string{"logs"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Tail, "tail", "t", true, "Tails the build log to the current terminal")
	cmd.Flags().BoolVarP(&options.Wait, "wait", "w", false, "Waits for the build to start before failing")
	cmd.Flags().DurationVarP(&options.WaitForPipelineDuration, "wait-duration", "d", time.Minute*5, "Timeout period waiting for the given pipeline to be created")
	cmd.Flags().BoolVarP(&options.BuildFilter.Pending, "pending", "p", false, "Only display logs which are currently pending to choose from if no build name is supplied")
	cmd.Flags().StringVarP(&options.BuildFilter.Filter, "filter", "f", "", "Filters all the available jobs by those that contain the given text")
	cmd.Flags().StringVarP(&options.BuildFilter.Owner, "owner", "o", "", "Filters the owner (person/organisation) of the repository")
	cmd.Flags().StringVarP(&options.BuildFilter.Repository, "repo", "r", "", "Filters the build repository")
	cmd.Flags().StringVarP(&options.BuildFilter.Branch, "branch", "", "", "Filters the branch")
	cmd.Flags().StringVarP(&options.BuildFilter.Build, "build", "", "", "The build number to view")
	cmd.Flags().StringVarP(&options.BuildFilter.Pod, "pod", "", "", "The pod name to view")
	cmd.Flags().StringVarP(&options.BuildFilter.Context, "context", "", "", "Filters the context of the build")
	cmd.Flags().BoolVarP(&options.CurrentFolder, "current", "c", false, "Display logs using current folder as repo name, and parent folder as owner")
	options.JenkinsSelector.AddFlags(cmd)
	options.AddBaseFlags(cmd)

	return cmd
}

// Run implements this command
func (o *GetBuildLogsOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
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

	tektonEnabled, err := kube.IsTektonEnabled(kubeClient, ns)
	if err != nil {
		return err
	}

	devEnv, err := kube.GetEnrichedDevEnvironment(kubeClient, jxClient, ns)
	if devEnv == nil {
		return fmt.Errorf("No development environment found for namespace %s", ns)
	}
	webhookEngine := devEnv.Spec.WebHookEngine
	if (webhookEngine == v1.WebHookEngineProw || webhookEngine == v1.WebHookEngineLighthouse) && !o.JenkinsSelector.IsCustom() {
		return o.getProwBuildLog(kubeClient, tektonClient, jxClient, ns, tektonEnabled)
	}

	args := o.Args

	if !o.BatchMode && len(args) == 0 {
		jobMap, err := o.GetJenkinsJobs(&o.JenkinsSelector, o.BuildFilter.Filter)
		if err != nil {
			return err
		}
		names := []string{}
		for k := range jobMap {
			names = append(names, k)
		}
		sort.Strings(names)
		if len(names) == 0 {
			return fmt.Errorf("No pipelines have been built!")
		}

		defaultName := ""
		for _, n := range names {
			if strings.HasSuffix(n, "/master") {
				defaultName = n
				break
			}
		}
		name, err := util.PickNameWithDefault(names, "Which pipeline do you want to view the logs of?: ", defaultName, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		args = []string{name}
	}
	if len(args) == 0 {
		return fmt.Errorf("No pipeline chosen")
	}
	name := args[0]
	buildNumber := o.BuildFilter.BuildNumber()

	last, err := o.getLastJenkinsBuild(name, buildNumber)
	if err != nil {
		return err
	}

	log.Logger().Infof("%s %s", util.ColorStatus("view the log at:"), util.ColorInfo(util.UrlJoin(last.Url, "/console")))
	return o.TailJenkinsBuildLog(&o.JenkinsSelector, name, &last)
}

func (o *GetBuildLogsOptions) getLastJenkinsBuild(name string, buildNumber int) (gojenkins.Build, error) {
	var last gojenkins.Build

	jenkinsClient, err := o.CreateCustomJenkinsClient(&o.JenkinsSelector)
	if err != nil {
		return last, err
	}

	f := func() error {
		var err error

		jobMap, err := o.GetJenkinsJobs(&o.JenkinsSelector, o.BuildFilter.Filter)
		if err != nil {
			return err
		}
		job := jobMap[name]
		if job.Url == "" {
			return fmt.Errorf("No Job exists yet called %s", name)
		}
		job.Url = jenkins.SwitchJenkinsBaseURL(job.Url, jenkinsClient.BaseURL())

		if buildNumber > 0 {
			last, err = jenkinsClient.GetBuild(job, buildNumber)
		} else {
			last, err = jenkinsClient.GetLastBuild(job)
		}
		if err != nil {
			return err
		}
		if last.Url == "" {
			if buildNumber > 0 {
				return fmt.Errorf("No build found for name %s number %d", name, buildNumber)
			} else {
				return fmt.Errorf("No build found for name %s", name)
			}
		}
		last.Url = jenkins.SwitchJenkinsBaseURL(last.Url, jenkinsClient.BaseURL())
		return err
	}

	if o.Wait {
		err := o.Retry(60, time.Second*2, f)
		return last, err
	} else {
		err := f()
		return last, err
	}
}

// getProwBuildLog prompts the user, if needed, to choose a pipeline, and then prints out that pipeline's logs.
func (o *GetBuildLogsOptions) getProwBuildLog(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, jxClient versioned.Interface, ns string, tektonEnabled bool) error {
	if o.CurrentFolder {
		currentDirectory, err := os.Getwd()
		if err != nil {
			return err
		}

		gitRepository, err := gits.NewGitCLI().Info(currentDirectory)
		if err != nil {
			return err
		}

		o.BuildFilter.Repository = gitRepository.Name
		o.BuildFilter.Owner = gitRepository.Organisation
	}

	var names []string
	var defaultName string
	var pipelineMap map[string][]builds.BaseBuildInfo

	var err error
	if tektonEnabled {
		var waitableCondition bool
		f := func() error {
			waitableCondition, err = o.getTektonLogs(kubeClient, tektonClient, jxClient, ns)
			return err
		}
		err = f()
		if err != nil {
			if o.Wait && waitableCondition {
				log.Logger().Info("The selected pipeline didn't start, let's wait a bit")
				err := util.Retry(o.WaitForPipelineDuration, f)
				if err != nil {
					return err
				}
			}
			return err
		}
		return nil
	}

	names, defaultName, pipelineMap, err = o.loadBuilds(kubeClient, ns)
	// If there's an error and we're waiting, ignore it.
	if err != nil && !o.Wait {
		return err
	}

	args := o.Args
	pickedPipeline := false
	if len(args) == 0 {
		if o.BatchMode {
			return util.MissingArgument("pipeline")
		}
		pickedPipeline = true
		name, err := util.PickNameWithDefault(names, "Which build do you want to view the logs of?: ", defaultName, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		args = []string{name}
	}
	if len(args) == 0 {
		return fmt.Errorf("No pipeline chosen")
	}
	name := args[0]
	if len(pipelineMap[name]) == 0 && !pickedPipeline && o.Wait {
		log.Logger().Infof("waiting for pipeline %s to start...", util.ColorInfo(name))

		// there's no pipeline with yet called 'name' so lets wait for it to start...
		f := func() error {
			var err error
			names, defaultName, pipelineMap, err = o.loadBuilds(kubeClient, ns)
			if err != nil {
				return err
			}
			if len(pipelineMap[name]) == 0 {
				// Look for pipelines that match the given name with a build number afterwards, and then set the name
				// to the most recent one, if there are any candidates.
				var candidateNames []string
				for k, v := range pipelineMap {
					if strings.HasPrefix(k, name+" #") && len(v) > 0 {
						candidateNames = append(candidateNames, k)
					}
				}
				if len(candidateNames) > 0 {
					sort.Slice(candidateNames, func(i, j int) bool {
						return buildNumberFromBaseBuildInfo(pipelineMap[candidateNames[i]][0]) > buildNumberFromBaseBuildInfo(pipelineMap[candidateNames[j]][0])
					})
					name = candidateNames[0]
					return nil
				}
				log.Logger().Infof("no build found in: %s", util.ColorInfo(strings.Join(names, ", ")))
				return fmt.Errorf("No pipeline exists yet: %s", name)
			}
			return nil
		}
		err := util.Retry(o.WaitForPipelineDuration, f)
		if err != nil {
			return err
		}
	}
	if len(pipelineMap[name]) == 0 {
		return fmt.Errorf("No Pipeline found for name %s in values: %s", name, strings.Join(names, ", "))
	}
	return o.getLogForKnative(kubeClient, ns, name, pipelineMap)
}

func (o *GetBuildLogsOptions) getLogForKnative(kubeClient kubernetes.Interface, ns string, name string, pipelineMap map[string][]builds.BaseBuildInfo) error {
	b := pipelineMap[name][0].(*builds.BuildPodInfo)
	pod := b.Pod
	if pod == nil {
		return fmt.Errorf("No Pod found for name %s", name)
	}
	containers, _, _ := kube.GetContainersWithStatusAndIsInit(pod)
	if len(containers) <= 0 {
		return fmt.Errorf("No Containers for Pod %s for build: %s", pod.Name, name)
	}

	log.Logger().Infof("Build logs for %s", util.ColorInfo(name))
	for i, ic := range containers {
		pod, err := kubeClient.CoreV1().Pods(ns).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to find pod %s", pod.Name)
		}
		if i > 0 {
			_, containerStatuses, _ := kube.GetContainersWithStatusAndIsInit(pod)
			if i < len(containerStatuses) {
				lastContainer := containerStatuses[i-1]
				terminated := lastContainer.State.Terminated
				if terminated != nil && terminated.ExitCode != 0 {
					log.Logger().Warnf("container %s failed with exit code %d: %s", lastContainer.Name, terminated.ExitCode, terminated.Message)
				}
			}
		}
		pod, err = waitForContainerToStart(kubeClient, ns, pod, i)
		if err != nil {
			return err
		}
		err = o.getPodLog(ns, pod, ic)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *GetBuildLogsOptions) getTektonLogs(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, jxClient versioned.Interface, ns string) (bool, error) {
	var defaultName string
	names, paMap, err := logs.GetTektonPipelinesWithActivePipelineActivity(jxClient, tektonClient, ns, o.BuildFilter.LabelSelectorsForActivity(), o.BuildFilter.Context)
	if err != nil {
		return true, err
	}

	var filter string
	if len(o.Args) > 0 {
		filter = o.Args[0]
	} else {
		filter = o.BuildFilter.Filter
	}

	var filteredNames []string
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), strings.ToLower(filter)) {
			filteredNames = append(filteredNames, n)
		}
	}

	if o.BatchMode {
		if len(filteredNames) > 1 {
			return false, errors.New("more than one pipeline returned in batch mode, use better filters and try again")
		}
		if len(filteredNames) == 1 {
			defaultName = filteredNames[0]
		}
	}

	name, err := util.PickNameWithDefault(filteredNames, "Which build do you want to view the logs of?: ", defaultName, "", o.In, o.Out, o.Err)
	if err != nil {
		return len(filteredNames) == 0, err
	}

	logWriter := CLILogWriter{
		o.CommonOptions,
	}

	pa, exists := paMap[name]
	if !exists {
		return true, errors.New("there are no build logs for the supplied filters")
	}

	if pa.Spec.BuildLogsURL != "" {
		return false, logs.StreamPipelinePersistentLogs(logWriter, pa.Spec.BuildLogsURL)
	}

	_ = logWriter.WriteLog(fmt.Sprintf("Build logs for %s", util.ColorInfo(name)))
	name = strings.TrimSuffix(name, " ")
	return false, logs.GetRunningBuildLogs(pa, name, kubeClient, tektonClient, logWriter)
}

// StreamLog implementation of LogWriter.StreamLog for CLILogWriter, this implementation will tail logs for the provided pod /container through the defined logger
func (o CLILogWriter) StreamLog(ns string, pod *corev1.Pod, container *corev1.Container) error {
	return o.TailLogs(ns, pod.Name, container.Name)
}

// WriteLog implementation of LogWriter.WriteLog for CLILogWriter, this implementation will write the provided log line through the defined logger
func (o CLILogWriter) WriteLog(line string) error {
	log.Logger().Info(line)
	return nil
}

func checkForStagePod(kubeClient kubernetes.Interface, ns string, pr *tektonv1alpha1.PipelineRun, pri *tekton.PipelineRunInfo, stage *tekton.StageInfo) error {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{
		pipeline.GroupName + pipeline.PipelineRunLabelKey: pri.PipelineRun,
	}})
	if err != nil {
		return err
	}
	podList, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return err
	}
	if err := stage.SetPodsForStageInfo(podList, pri.PipelineRun); err != nil {
		return err
	}

	if stage.Pod == nil {
		// If we haven't found a pod for this stage and the pipeline has failed, log and return nil.
		if pr.Status.GetCondition(apis.ConditionSucceeded).IsFalse() {
			log.Logger().Warnf("no pod created for stage %s in build %s due to earlier failure", util.ColorInfo(stage.Name), util.ColorInfo(pri.PipelineRun))
			return nil
		}
		log.Logger().Infof("no pod found yet for stage %s in build %s", util.ColorInfo(stage.Name), util.ColorInfo(pri.PipelineRun))
		return fmt.Errorf("No pod for stage %s in build %s exists yet", stage.Name, pri.PipelineRun)
	}

	return nil
}

func buildNumberFromBaseBuildInfo(info builds.BaseBuildInfo) int {
	n, err := strconv.Atoi(info.GetBuild())
	if err != nil {
		// If there's an error, just fall back on 0 so this gets ranked last.
		return 0
	}
	return n
}

func waitForContainerToStart(kubeClient kubernetes.Interface, ns string, pod *corev1.Pod, idx int) (*corev1.Pod, error) {
	if pod.Status.Phase == corev1.PodFailed {
		log.Logger().Warnf("pod %s has failed", pod.Name)
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
	log.Logger().Infof("waiting for pod %s container %s to start...", util.ColorInfo(pod.Name), util.ColorInfo(containerName))
	for {
		time.Sleep(time.Second)

		p, err := kubeClient.CoreV1().Pods(ns).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return p, errors.Wrapf(err, "failed to load pod %s", pod.Name)
		}
		if kube.HasContainerStarted(p, idx) {
			return p, nil
		}
	}
}

func (o *GetBuildLogsOptions) getPodLog(ns string, pod *corev1.Pod, container corev1.Container) error {
	log.Logger().Infof("getting the log for pod %s and container %s", util.ColorInfo(pod.Name), util.ColorInfo(container.Name))
	return o.TailLogs(ns, pod.Name, container.Name)
}

func (o *GetBuildLogsOptions) getStageLog(ns, build, stageName string, pod *corev1.Pod, container corev1.Container) error {
	log.Logger().Infof("getting the log for build %s stage %s and container %s", util.ColorInfo(build), util.ColorInfo(stageName), util.ColorInfo(container.Name))
	return o.TailLogs(ns, pod.Name, container.Name)
}

func (o *GetBuildLogsOptions) loadBuilds(kubeClient kubernetes.Interface, ns string) ([]string, string, map[string][]builds.BaseBuildInfo, error) {
	defaultName := ""
	names := []string{}
	buildMap := map[string][]builds.BaseBuildInfo{}

	pods, err := builds.GetBuildPods(kubeClient, ns)
	if err != nil {
		log.Logger().Warnf("Failed to query pods %s", err)
		return names, defaultName, buildMap, err
	}

	buildInfos := []*builds.BuildPodInfo{}
	for _, pod := range pods {
		containers, _, _ := kube.GetContainersWithStatusAndIsInit(pod)
		if len(containers) > 0 {
			buildInfo := builds.CreateBuildPodInfo(pod)
			if o.BuildFilter.BuildMatches(buildInfo) {
				buildInfos = append(buildInfos, buildInfo)
			}
		}
	}
	builds.SortBuildPodInfos(buildInfos)
	if len(buildInfos) == 0 {
		return names, defaultName, buildMap, fmt.Errorf("no knative builds have been triggered which match the current filter")
	}

	for _, build := range buildInfos {
		name := build.Pipeline + " #" + build.Build
		names = append(names, name)
		buildMap[name] = append(buildMap[name], build)

		if build.Branch == "master" {
			defaultName = name
		}
	}
	return names, defaultName, buildMap, nil
}

// loadPipelines loads all available pipelines as PipelineRunInfos.
func (o *GetBuildLogsOptions) loadPipelines(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, jxClient versioned.Interface, ns string) ([]string, string, map[string][]builds.BaseBuildInfo, error) {
	defaultName := ""
	names := []string{}
	pipelineMap := map[string][]builds.BaseBuildInfo{}

	labelSelectors := o.BuildFilter.LabelSelectorsForBuild()

	listOptions := metav1.ListOptions{}
	if len(labelSelectors) > 0 {
		listOptions.LabelSelector = strings.Join(labelSelectors, ",")
	}

	prList, err := tektonClient.TektonV1alpha1().PipelineRuns(ns).List(listOptions)
	if err != nil {
		log.Logger().Warnf("Failed to query PipelineRuns %s", err)
		return names, defaultName, pipelineMap, err
	}

	structures, err := jxClient.JenkinsV1().PipelineStructures(ns).List(listOptions)
	if err != nil {
		log.Logger().Warnf("Failed to query PipelineStructures %s", err)
		return names, defaultName, pipelineMap, err
	}
	// TODO: Remove this eventually - it's only here for structures created before we started applying labels to them in v2.0.216.
	if len(prList.Items) > len(structures.Items) && len(labelSelectors) != 0 {
		structures, err = jxClient.JenkinsV1().PipelineStructures(ns).List(metav1.ListOptions{})
		if err != nil {
			log.Logger().Warnf("Failed to query PipelineStructures %s", err)
			return names, defaultName, pipelineMap, err
		}
	}

	buildInfos := []*tekton.PipelineRunInfo{}

	podLabelSelector := pipeline.GroupName + pipeline.PipelineRunLabelKey
	if len(labelSelectors) > 0 {
		podLabelSelector += "," + strings.Join(labelSelectors, ",")
	}
	podList, err := kubeClient.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: podLabelSelector,
	})
	if err != nil {
		return names, defaultName, pipelineMap, err
	}
	for _, pr := range prList.Items {
		var ps v1.PipelineStructure
		for _, p := range structures.Items {
			if p.Name == pr.Name {
				ps = p
			}
		}
		pri, err := tekton.CreatePipelineRunInfo(pr.Name, podList, &ps, &pr)
		if err != nil {
			log.Logger().Warnf("Error creating PipelineRunInfo for PipelineRun %s: %s", pr.Name, err)
		}
		if pri != nil && o.BuildFilter.BuildMatches(pri.ToBuildPodInfo()) {
			buildInfos = append(buildInfos, pri)
		}
	}

	tekton.SortPipelineRunInfos(buildInfos)
	if len(buildInfos) == 0 {
		return names, defaultName, pipelineMap, fmt.Errorf("no Tekton pipelines have been triggered which match the current filter")
	}

	namesMap := make(map[string]bool, 0)
	for _, build := range buildInfos {
		buildName := build.Pipeline + " #" + build.Build
		if build.Context != "" {
			buildName += " " + build.Context
		}
		namesMap[buildName] = true
		pipelineMap[buildName] = append(pipelineMap[buildName], build)

		if build.Branch == "master" {
			defaultName = buildName
		}
	}
	for k := range namesMap {
		names = append(names, k)
	}

	return names, defaultName, pipelineMap, nil
}

func (o *GetBuildLogsOptions) loadPipelineActivities(jxClient versioned.Interface, ns string) (*v1.PipelineActivityList, error) {
	paList, err := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "there was a problem getting the PipelineActivities")
	}

	return paList, nil
}
