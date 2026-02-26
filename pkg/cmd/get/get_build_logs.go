package get

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/v2/pkg/builds"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/logs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/util"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
)

// GetBuildLogsOptions the command line options
type GetBuildLogsOptions struct {
	Options

	Tail                    bool
	Wait                    bool
	BuildFilter             builds.BuildPodInfoFilter
	CurrentFolder           bool
	WaitForPipelineDuration time.Duration
	TektonLogger            *logs.TektonLogger
	FailIfPodFails          bool
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

		# Pick a pending Tekton build to view the log based 
		jx get build log -p

		# Pick a pending Tekton build to view the log based on the repo cheese
		jx get build log --repo cheese -p

		# Pick a Tekton build for the 1234 Pull Request on the repo cheese
		jx get build log --repo cheese --branch PR-1234

		# View the build logs for a specific tekton build pod
		jx get build log --pod my-pod-name
	`)
)

// NewCmdGetBuildLogs creates the command
func NewCmdGetBuildLogs(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetBuildLogsOptions{
		Options: Options{
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
	cmd.Flags().BoolVarP(&options.FailIfPodFails, "fail-with-pod", "", false, "Return an error if the pod fails")
	cmd.Flags().DurationVarP(&options.WaitForPipelineDuration, "wait-duration", "d", time.Minute*5, "Timeout period waiting for the given pipeline to be created")
	cmd.Flags().BoolVarP(&options.BuildFilter.Pending, "pending", "p", false, "Only display logs which are currently pending to choose from if no build name is supplied")
	cmd.Flags().StringVarP(&options.BuildFilter.Filter, "filter", "f", "", "Filters all the available jobs by those that contain the given text")
	cmd.Flags().StringVarP(&options.BuildFilter.Owner, "owner", "o", "", "Filters the owner (person/organisation) of the repository")
	cmd.Flags().StringVarP(&options.BuildFilter.Repository, "repo", "r", "", "Filters the build repository")
	cmd.Flags().StringVarP(&options.BuildFilter.Branch, "branch", "", "", "Filters the branch")
	cmd.Flags().StringVarP(&options.BuildFilter.Build, "build", "", "", "The build number to view")
	cmd.Flags().StringVarP(&options.BuildFilter.Pod, "pod", "", "", "The pod name to view")
	cmd.Flags().StringVarP(&options.BuildFilter.GitURL, "giturl", "g", "", "The git URL to filter on. If you specify a link to a github repository or PR we can filter the query of build pods accordingly")
	cmd.Flags().StringVarP(&options.BuildFilter.Context, "context", "", "", "Filters the context of the build")
	cmd.Flags().BoolVarP(&options.CurrentFolder, "current", "c", false, "Display logs using current folder as repo name, and parent folder as owner")
	options.AddBaseFlags(cmd)

	return cmd
}

// Run implements this command
func (o *GetBuildLogsOptions) Run() error {
	err := o.BuildFilter.Validate()
	if err != nil {
		return err
	}
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

	return o.getProwBuildLog(kubeClient, tektonClient, jxClient, ns, tektonEnabled)
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

	var err error

	if o.TektonLogger == nil {
		o.TektonLogger = &logs.TektonLogger{
			KubeClient:     kubeClient,
			TektonClient:   tektonClient,
			JXClient:       jxClient,
			Namespace:      ns,
			FailIfPodFails: o.FailIfPodFails,
		}
	}
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

func (o *GetBuildLogsOptions) getTektonLogs(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, jxClient versioned.Interface, ns string) (bool, error) {
	var defaultName string

	names, paMap, err := o.TektonLogger.GetTektonPipelinesWithActivePipelineActivity(o.BuildFilter.LabelSelectorsForActivity())
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

	name, err := util.PickNameWithDefault(filteredNames, "Which build do you want to view the logs of?: ", defaultName, "", o.GetIOFileHandles())
	if err != nil {
		return len(filteredNames) == 0, err
	}

	pa, exists := paMap[name]
	if !exists {
		return true, errors.New("there are no build logs for the supplied filters")
	}

	if pa.Spec.BuildLogsURL != "" {
		authSvc, err := o.GitAuthConfigService()
		if err != nil {
			return false, err
		}
		for line := range o.TektonLogger.StreamPipelinePersistentLogs(pa.Spec.BuildLogsURL, authSvc) {
			fmt.Fprintln(o.Out, line.Line)
		}
		return false, o.TektonLogger.Err()
	}

	log.Logger().Infof("Build logs for %s", util.ColorInfo(name))
	name = strings.TrimSuffix(name, " ")
	for line := range o.TektonLogger.GetRunningBuildLogs(pa, name, false) {
		fmt.Fprintln(o.Out, line.Line)
	}
	return false, o.TektonLogger.Err()
}
