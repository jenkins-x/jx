package get

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
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
	if webhookEngine == v1.WebHookEngineProw && !o.JenkinsSelector.IsCustom() {
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
	var buildMap map[string]builds.BaseBuildInfo
	var pipelineMap map[string]builds.BaseBuildInfo

	args := o.Args
	pickedPipeline := false
	if len(args) == 0 {
		if o.BatchMode {
			return util.MissingArgument("pipeline")
		}
		var err error
		if tektonEnabled {
			names, defaultName, buildMap, pipelineMap, err = o.loadPipelines(kubeClient, tektonClient, jxClient, ns)
		} else {
			names, defaultName, buildMap, pipelineMap, err = o.loadBuilds(kubeClient, ns)
		}
		if err != nil {
			return err
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
	build := buildMap[name]
	suffix := ""
	if build == nil {
		build = pipelineMap[name]
		if build != nil {
			suffix = " #" + build.GetBuild()
		}
	}
	if build == nil && !pickedPipeline && o.Wait {
		log.Logger().Infof("waiting for pipeline %s to start...", util.ColorInfo(name))

		// there's no pipeline with yet called 'name' so lets wait for it to start...
		f := func() error {
			var err error
			if tektonEnabled {
				names, defaultName, buildMap, pipelineMap, err = o.loadPipelines(kubeClient, tektonClient, jxClient, ns)
			} else {
				names, defaultName, buildMap, pipelineMap, err = o.loadBuilds(kubeClient, ns)
			}
			if err != nil {
				return err
			}
			build = buildMap[name]
			if build == nil {
				build = pipelineMap[name]
				if build != nil {
					suffix = " #" + build.GetBuild()
				}
			}
			if build == nil {
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
	if build == nil {
		return fmt.Errorf("No Pipeline found for name %s in values: %s", name, strings.Join(names, ", "))
	}

	if tektonEnabled {
		pr := build.(*tekton.PipelineRunInfo)
		log.Logger().Infof("Build logs for %s", util.ColorInfo(name+suffix))
		for _, stage := range pr.GetOrderedTaskStages() {
			if stage.Pod == nil {
				// The stage's pod hasn't been created yet, so let's wait a bit.
				f := func() error {
					selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{
						pipeline.GroupName + pipeline.PipelineRunLabelKey: pr.PipelineRun,
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
					if err := stage.SetPodsForStageInfo(podList, pr.PipelineRun); err != nil {
						return err
					}

					if stage.Pod == nil {
						log.Logger().Infof("no pod found yet for stage %s in build %s", util.ColorInfo(stage.Name), util.ColorInfo(pr.PipelineRun))
						return fmt.Errorf("No pod for stage %s in build %s exists yet", stage.Name, pr.PipelineRun)
					}

					return nil
				}
				err := util.Retry(o.WaitForPipelineDuration, f)
				if err != nil {
					return err
				}
			}
			pod := stage.Pod
			containers, _, _ := kube.GetContainersWithStatusAndIsInit(pod)
			if len(containers) <= 0 {
				return fmt.Errorf("No Containers for Pod %s for build: %s", pod.Name, name)
			}
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
				err = o.getStageLog(ns, name+suffix, stage.GetStageNameIncludingParents(), pod, ic)
				if err != nil {
					return err
				}
			}
		}
	} else {
		b := build.(*builds.BuildPodInfo)
		pod := b.Pod
		if pod == nil {
			return fmt.Errorf("No Pod found for name %s", name)
		}
		containers, _, _ := kube.GetContainersWithStatusAndIsInit(pod)
		if len(containers) <= 0 {
			return fmt.Errorf("No Containers for Pod %s for build: %s", pod.Name, name)
		}

		log.Logger().Infof("Build logs for %s", util.ColorInfo(name+suffix))
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
	}
	return nil
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

func (o *GetBuildLogsOptions) loadBuilds(kubeClient kubernetes.Interface, ns string) ([]string, string, map[string]builds.BaseBuildInfo, map[string]builds.BaseBuildInfo, error) {
	defaultName := ""
	names := []string{}
	buildMap := map[string]builds.BaseBuildInfo{}
	pipelineMap := map[string]builds.BaseBuildInfo{}

	pods, err := builds.GetBuildPods(kubeClient, ns)
	if err != nil {
		log.Logger().Warnf("Failed to query pods %s", err)
		return names, defaultName, buildMap, pipelineMap, err
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
		return names, defaultName, buildMap, pipelineMap, fmt.Errorf("no knative builds have been triggered which match the current filter")
	}

	for _, build := range buildInfos {
		name := build.Pipeline + " #" + build.Build
		names = append(names, name)
		buildMap[name] = build
		pipelineMap[build.Pipeline] = build

		if build.Branch == "master" {
			defaultName = name
		}
	}
	return names, defaultName, buildMap, pipelineMap, nil
}

func (o *GetBuildLogsOptions) loadPipelines(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, jxClient versioned.Interface, ns string) ([]string, string, map[string]builds.BaseBuildInfo, map[string]builds.BaseBuildInfo, error) {
	defaultName := ""
	names := []string{}
	buildMap := map[string]builds.BaseBuildInfo{}
	pipelineMap := map[string]builds.BaseBuildInfo{}

	labelSelectors := o.BuildFilter.LabelSelectorsForBuild()

	listOptions := metav1.ListOptions{}
	if len(labelSelectors) > 0 {
		listOptions.LabelSelector = strings.Join(labelSelectors, ",")
	}

	prList, err := tektonClient.TektonV1alpha1().PipelineRuns(ns).List(listOptions)
	if err != nil {
		log.Logger().Warnf("Failed to query PipelineRuns %s", err)
		return names, defaultName, buildMap, pipelineMap, err
	}

	structures, err := jxClient.JenkinsV1().PipelineStructures(ns).List(listOptions)
	if err != nil {
		log.Logger().Warnf("Failed to query PipelineStructures %s", err)
		return names, defaultName, buildMap, pipelineMap, err
	}
	// TODO: Remove this eventually - it's only here for structures created before we started applying labels to them.
	if len(prList.Items) > len(structures.Items) && len(labelSelectors) != 0 {
		structures, err = jxClient.JenkinsV1().PipelineStructures(ns).List(metav1.ListOptions{})
		if err != nil {
			log.Logger().Warnf("Failed to query PipelineStructures %s", err)
			return names, defaultName, buildMap, pipelineMap, err
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
		return names, defaultName, buildMap, pipelineMap, err
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
		return names, defaultName, buildMap, pipelineMap, fmt.Errorf("no Tekton pipelines have been triggered which match the current filter")
	}

	for _, build := range buildInfos {
		name := build.Pipeline + " #" + build.Build
		if build.Context != "" {
			name += " " + build.Context
		}
		names = append(names, name)
		buildMap[name] = build
		pipelineMap[build.Pipeline] = build

		if build.Branch == "master" {
			defaultName = name
		}
	}
	return names, defaultName, buildMap, pipelineMap, nil
}
