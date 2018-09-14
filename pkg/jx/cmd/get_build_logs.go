package cmd

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetBuildLogsOptions the command line options
type GetBuildLogsOptions struct {
	GetOptions

	Tail   bool
	Filter string
	Build  int
}

var (
	get_build_log_long = templates.LongDesc(`
		Display the git server URLs.

`)

	get_build_log_example = templates.Examples(`
		# List all registered git server URLs
		jx get git
	`)
)

// NewCmdGetBuildLogs creates the command
func NewCmdGetBuildLogs(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetBuildLogsOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,

				Out: out,
				Err: errOut,
			},
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
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Tail, "tail", "t", true, "Tails the build log to the current terminal")
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filters all the available jobs by those that contain the given text")
	cmd.Flags().IntVarP(&options.Build, "build", "b", 0, "The build number to view")

	return cmd
}

// Run implements this command
func (o *GetBuildLogsOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	devEnv, err := kube.GetEnrichedDevEnvironment(kubeClient, jxClient, ns)
	webhookEngine := devEnv.Spec.WebHookEngine
	if webhookEngine == v1.WebHookEngineProw {
		return o.getProwBuildLog(kubeClient, jxClient, ns)
	}
	jobMap, err := o.getJobMap(o.Filter)
	if err != nil {
		return err
	}
	jenkinsClient, err := o.JenkinsClient()
	if err != nil {
		return err
	}

	args := o.Args
	names := []string{}
	for k, _ := range jobMap {
		names = append(names, k)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("No pipelines have been built!")
	}

	if len(args) == 0 {
		defaultName := ""
		for _, n := range names {
			if strings.HasSuffix(n, "/master") {
				defaultName = n
				break
			}
		}
		name, err := util.PickNameWithDefault(names, "Which pipeline do you want to view the logs of?: ", defaultName, o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		args = []string{name}
	}
	if len(args) == 0 {
		return fmt.Errorf("No pipeline chosen")
	}
	name := args[0]
	job := jobMap[name]
	var last gojenkins.Build
	if o.Build > 0 {
		last, err = jenkinsClient.GetBuild(job, o.Build)
	} else {
		last, err = jenkinsClient.GetLastBuild(job)
	}
	if err != nil {
		return err
	}
	log.Infof("%s %s\n", util.ColorStatus("view the log at:"), util.ColorInfo(util.UrlJoin(last.Url, "/console")))
	return o.tailBuild(name, &last)
}

func (o *GetBuildLogsOptions) getProwBuildLog(kubeClient kubernetes.Interface, jxClient versioned.Interface, ns string) error {
	activities := jxClient.JenkinsV1().PipelineActivities(ns)
	pipelineList, err := activities.List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	args := o.Args
	names := []string{}
	pipelineMap := map[string]map[int]*v1.PipelineActivity{}

	defaultName := ""
	for _, activity := range pipelineList.Items {
		pipeline := activity.Spec.Pipeline
		build := activity.Spec.Build
		if defaultName == "" && strings.HasSuffix(pipeline, "/master") {
			defaultName = pipeline
		}
		if pipeline == "" || build == "" {
			continue
		}
		buildNumber, err := strconv.Atoi(build)
		if err != nil {
			continue
		}
		if util.StringArrayIndex(names, pipeline) < 0 {
			names = append(names, pipeline)
		}
		copy := activity
		buildMap := pipelineMap[pipeline]
		if buildMap == nil {
			buildMap = map[int]*v1.PipelineActivity{}
		}
		buildMap[buildNumber] = &copy
		pipelineMap[pipeline] = buildMap
	}

	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("No pipelines have been triggered!")
	}

	if len(args) == 0 {
		name, err := util.PickNameWithDefault(names, "Which pipeline do you want to view the logs of?: ", defaultName, o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		args = []string{name}
	}
	if len(args) == 0 {
		return fmt.Errorf("No pipeline chosen")
	}
	name := args[0]
	buildMap := pipelineMap[name]
	if buildMap == nil {
		return fmt.Errorf("No Pipeline found for name %s", name)
	}
	var build *v1.PipelineActivity

	buildNumber := o.Build
	if buildNumber > 0 {
		build = buildMap[buildNumber]
		if build == nil {
			return fmt.Errorf("No Pipeline found for %s and build #%d", name, buildNumber)
		}
	} else {
		for k, v := range buildMap {
			if k > buildNumber {
				buildNumber = k
				build = v
			}
		}
	}
	if build == nil {
		return fmt.Errorf("No Pipeline builds found for %s", name)
	}
	if err != nil {
		return err
	}
	log.Infof("Getting the log of pipeline %s build %s\n", util.ColorInfo(name), util.ColorInfo("#"+strconv.Itoa(buildNumber)))

	pods, err := builds.GetBuildPods(kubeClient, ns)
	if err != nil {
		return err
	}

	for _, pod := range pods {
		log.Infof("found pod %s\n", pod.Name)
		initContainers := pod.Spec.InitContainers
		if len(initContainers) > 0 {
			lastInitC := initContainers[len(initContainers)-1]
			params := BuildParams{}
			params.DefaultValuesFromEnvVars(lastInitC.Env)

			if params.MatchesPipeline(build) {
				return o.getPodLog(ns, pod, lastInitC)
			}
		}
	}
	log.Warnf("No pod is available for pipeline %s build %s\n", util.ColorInfo(name), util.ColorInfo("#"+strconv.Itoa(buildNumber)))
	return nil
}

func (o *GetBuildLogsOptions) getPodLog(ns string, pod *corev1.Pod, container corev1.Container) error {
	log.Infof("Getting the pod log for pod %s and init container %s\n", pod.Name, container.Name)
	return o.tailLogs(ns, pod.Name, container.Name)
}

type BuildParams struct {
	GitOwner      string
	GitRepository string
	BranchName    string
	BuildNumber   string
}

// DefaultValuesFromEnvVars defaults values from the environment variables
func (p *BuildParams) DefaultValuesFromEnvVars(envVars []corev1.EnvVar) {
	for _, buildNumberKey := range []string{"JX_BUILD_NUMBER", "BUILD_NUMBER", "BUILD_ID"} {
		for _, env := range envVars {
			value := env.Value
			switch env.Name {
			case "BRANCH_NAME":
				p.BranchName = value
			case "REPO_NAME":
				p.GitRepository = value
			case "REPO_OWNER":
				p.GitOwner = value
			case buildNumberKey:
				if p.BuildNumber == "" {
					p.BuildNumber = value
				}
			}
		}
	}
}

// MatchesPipeline returns true if the given pipeline matches the build parameters
func (p *BuildParams) MatchesPipeline(activity *v1.PipelineActivity) bool {
	if p.GitOwner == "" || p.GitRepository == "" || p.BranchName == "" || p.BuildNumber == "" {
		return false
	}
	d := kube.CreatePipelineDetails(activity)
	if d == nil {
		return false
	}
	return d.GitOwner == p.GitOwner && d.GitRepository == p.GitRepository && d.Build == p.BuildNumber && d.BranchName == p.BranchName
}
