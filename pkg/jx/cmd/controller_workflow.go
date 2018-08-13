package cmd

import (
	"io"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	"github.com/jenkins-x/jx/pkg/kube"
)

// ControllerWorkflowOptions are the flags for the commands
type ControllerWorkflowOptions struct {
	ControllerOptions

	Namespace string
	NoWatch   bool

	workflowMap map[string]*v1.Workflow
}

// NewCmdControllerWorkflow creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdControllerWorkflow(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ControllerWorkflowOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Runs the workflow controller",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		Aliases: []string{"workflows"},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to watch or defaults to the current namespace")
	cmd.Flags().BoolVarP(&options.NoWatch, "no-watch", "", false, "Disable watch so just performs any delta processes on pending workflows")
	return cmd
}

// Run implements this command
func (o *ControllerWorkflowOptions) Run() error {
	err := o.registerWorkflowCRD()
	if err != nil {
		return err
	}
	err = o.registerWorkflowCRD()
	if err != nil {
		return err
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = devNs
	}

	o.workflowMap = map[string]*v1.Workflow{}

	if o.NoWatch {
		return o.updatePipelinesWithoutWatching(jxClient, ns)
	}

	log.Infof("Watching for PipelineActivity resources in namespace %s\n", util.ColorInfo(ns))
	workflow := &v1.Workflow{}
	activity := &v1.PipelineActivity{}
	workflowListWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "workflows", ns, fields.Everything())
	kube.SortListWatchByName(workflowListWatch)
	_, workflowController := cache.NewInformer(
		workflowListWatch,
		workflow,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onWorkflowObj(obj, jxClient, ns)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onWorkflowObj(newObj, jxClient, ns)
			},
			DeleteFunc: func(obj interface{}) {
				o.deleteWorkflowObjb(obj, jxClient, ns)
			},
		},
	)
	stop := make(chan struct{})
	go workflowController.Run(stop)

	pipelineListWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "pipelineactivities", ns, fields.Everything())
	kube.SortListWatchByName(pipelineListWatch)
	_, pipelineController := cache.NewInformer(
		pipelineListWatch,
		activity,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onActivityObj(obj, jxClient, ns)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onActivityObj(newObj, jxClient, ns)
			},
			DeleteFunc: func(obj interface{}) {
			},
		},
	)

	go pipelineController.Run(stop)

	// Wait forever
	select {}
}

func (o *ControllerWorkflowOptions) updatePipelinesWithoutWatching(jxClient versioned.Interface, ns string) error {
	workflows, err := jxClient.JenkinsV1().Workflows(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, workflow := range workflows.Items {
		o.onWorkflow(&workflow, jxClient, ns)
	}

	pipelines, err := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pipeline := range pipelines.Items {
		o.onActivity(&pipeline, jxClient, ns)
	}
	return nil
}

func (o *ControllerWorkflowOptions) onWorkflowObj(obj interface{}, jxClient versioned.Interface, ns string) {
	workflow, ok := obj.(*v1.Workflow)
	if !ok {
		log.Infof("Object is not a Workflow %#v\n", obj)
		return
	}
	if workflow != nil {
		o.onWorkflow(workflow, jxClient, ns)
	}
}

func (o *ControllerWorkflowOptions) deleteWorkflowObjb(obj interface{}, jxClient versioned.Interface, ns string) {
	workflow, ok := obj.(*v1.Workflow)
	if !ok {
		log.Infof("Object is not a Workflow %#v\n", obj)
		return
	}
	if workflow != nil {
		o.onWorkflowDelete(workflow, jxClient, ns)
	}
}

func (o *ControllerWorkflowOptions) onWorkflow(workflow *v1.Workflow, jxClient versioned.Interface, ns string) {
	o.workflowMap[workflow.Name] = workflow
}

func (o *ControllerWorkflowOptions) onWorkflowDelete(workflow *v1.Workflow, jxClient versioned.Interface, ns string) {
	delete(o.workflowMap, workflow.Name)
}

func (o *ControllerWorkflowOptions) onActivityObj(obj interface{}, jxClient versioned.Interface, ns string) {
	pipeline, ok := obj.(*v1.PipelineActivity)
	if !ok {
		log.Infof("Object is not a PipelineActivity %#v\n", obj)
		return
	}
	if pipeline != nil {
		o.onActivity(pipeline, jxClient, ns)
	}
}

func (o *ControllerWorkflowOptions) onActivity(pipeline *v1.PipelineActivity, jxClient versioned.Interface, ns string) {
	workflowName := pipeline.Spec.Workflow
	version := pipeline.Spec.Version
	repoName := pipeline.Spec.GitRepository
	pipelineName := pipeline.Spec.Pipeline
	build := pipeline.Spec.Build

	paths := strings.Split(pipelineName, "/")
	branch := paths[len(paths)-1]
	if repoName == "" && len(paths) > 1 {
		repoName = paths[len(paths)-2]
	}

	log.Infof("Processing pipeline %s repo %s version %s with workflow %s and status %s\n", pipeline.Name, repoName, version, workflowName, string(pipeline.Spec.WorkflowStatus))

	if workflowName == "" {
		workflowName = "default"
	}
	if repoName == "" || version == "" || build == "" || pipelineName == "" {
		log.Infof("Ignoring missing data for pipeline: %s repo: %s version: %s status: %s\n", pipeline.Name, repoName, version, string(pipeline.Spec.WorkflowStatus))
		return
	}
	if !pipeline.Spec.WorkflowStatus.IsTerminated() {
		flow := o.workflowMap[workflowName]
		if flow == nil && workflowName == "default" {
			var err error
			flow, err = workflow.CreateDefaultWorkflow(jxClient, ns)
			if err != nil {
				log.Warnf("Cannot create default Workflow: %s\n", err)
				return
			}

		}

		if flow == nil {
			log.Warnf("Cannot process pipeline %s due to workflow name %s not existing\n", pipeline.Name, workflowName)
			return
		}

		if !o.isReleaseBranch(branch) {
			log.Infof("Ignoring branch %s\n", branch)
			return
		}

		// lets walk the Workflow spec and see if we need to trigger any PRs or move the PipelineActivity forward
		promoteStatusMap := createPromoteStatus(pipeline)

		for _, step := range flow.Spec.Steps {
			promote := step.Promote
			if promote != nil {
				envName := promote.Environment
				if envName != "" {
					status := promoteStatusMap[envName]
					if status == nil || status.PullRequest == nil || status.PullRequest.PullRequestURL == "" {
						// can we generate a PR now?
						if canExecuteStep(flow, pipeline, &step, promoteStatusMap) {
							log.Infof("Creating PR for environment %s\n", envName)
							po := &PromoteOptions{
								Application:       repoName,
								Environment:       envName,
								Pipeline:          pipelineName,
								Build:             build,
								Version:           version,
								NoPoll:            true,
								IgnoreLocalFiles:  true,
								HelmRepositoryURL: helm.DefaultHelmRepositoryURL,
								LocalHelmRepoName: kube.LocalHelmRepoName,
							}
							po.CommonOptions = o.CommonOptions
							po.BatchMode = true

							err := po.Run()
							if err != nil {
								log.Warnf("Failed to create PullRequest on pipeline %s repo %s version %s with workflow %s: %s\n", pipeline.Name, repoName, version, workflowName, err)
							}
						}
					}
				}
			}
		}
	}
}

func canExecuteStep(workflow *v1.Workflow, activity *v1.PipelineActivity, step *v1.WorkflowStep, statusMap map[string]*v1.PromoteActivityStep) bool {
	for _, envName := range step.Preconditions.Environments {
		status := statusMap[envName]
		if status == nil || status.Status != v1.ActivityStatusTypeSucceeded {
			return false
		}
	}
	return true
}

// createPromoteStatus returns a map indexed by environment name of all the promotions in this pipeline
func createPromoteStatus(pipeline *v1.PipelineActivity) map[string]*v1.PromoteActivityStep {
	answer := map[string]*v1.PromoteActivityStep{}
	for _, step := range pipeline.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			envName := promote.Environment
			if envName != "" {
				answer[envName] = promote
			}
		}
	}
	return answer
}

// createPromoteStepActivityKey deduces the pipeline metadata from the knative workflow pod
func (o *ControllerWorkflowOptions) createPromoteStepActivityKey(buildName string, pod *corev1.Pod) *kube.PromoteStepActivityKey {
	branch := ""
	lastCommitSha := ""
	lastCommitMessage := ""
	lastCommitURL := ""
	build := digitSuffix(buildName)
	if build == "" {
		build = "1"
	}
	gitUrl := ""
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == "workflow-step-git-source" {
			args := initContainer.Args
			for i := 0; i <= len(args)-2; i += 2 {
				key := args[i]
				value := args[i+1]

				switch key {
				case "-url":
					gitUrl = value
				case "-revision":
					branch = value
				}
			}
			break
		}
	}
	if gitUrl == "" {
		return nil
	}
	if branch == "" {
		branch = "master"
	}
	gitInfo, err := gits.ParseGitURL(gitUrl)
	if err != nil {
		log.Warnf("Failed to parse git URL %s: %s", gitUrl, err)
		return nil
	}
	org := gitInfo.Organisation
	repo := gitInfo.Name
	name := org + "-" + repo + "-" + branch + "-" + build
	pipeline := org + "/" + repo + "/" + branch
	return &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:              name,
			Pipeline:          pipeline,
			Build:             build,
			LastCommitSHA:     lastCommitSha,
			LastCommitMessage: lastCommitMessage,
			LastCommitURL:     lastCommitURL,
			GitInfo:           gitInfo,
		},
	}
}

func (o *ControllerWorkflowOptions) isReleaseBranch(branchName string) bool {
	// TODO look in TeamSettings for a list of allowed release branch patterns
	return branchName == "master"

}
