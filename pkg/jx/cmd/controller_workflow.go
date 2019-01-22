package cmd

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	"github.com/jenkins-x/jx/pkg/kube"
)

// ControllerWorkflowOptions are the flags for the commands
type ControllerWorkflowOptions struct {
	ControllerOptions

	Namespace           string
	NoWatch             bool
	NoMergePullRequest  bool
	Verbose             bool
	LocalHelmRepoName   string
	PullRequestPollTime string

	// testing
	FakePullRequests CreateEnvPullRequestFn
	FakeGitProvider  *gits.FakeProvider

	// calculated fields
	PullRequestPollDuration *time.Duration
	workflowMap             map[string]*v1.Workflow
	pipelineMap             map[string]*v1.PipelineActivity
}

// NewCmdControllerWorkflow creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdControllerWorkflow(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &ControllerWorkflowOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
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

	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to watch or defaults to the current namespace")
	cmd.Flags().StringVarP(&options.LocalHelmRepoName, "helm-repo-name", "r", kube.LocalHelmRepoName, "The name of the helm repository that contains the app")
	cmd.Flags().BoolVarP(&options.NoWatch, "no-watch", "", false, "Disable watch so just performs any delta processes on pending workflows")
	cmd.Flags().StringVarP(&options.PullRequestPollTime, optionPullRequestPollTime, "", "20s", "Poll time when waiting for a Pull Request to merge")
	return cmd
}

// Run implements this command
func (o *ControllerWorkflowOptions) Run() error {
	err := o.registerPipelineActivityCRD()
	if err != nil {
		return err
	}
	err = o.registerWorkflowCRD()
	if err != nil {
		return err
	}

	if o.PullRequestPollTime != "" {
		duration, err := time.ParseDuration(o.PullRequestPollTime)
		if err != nil {
			return fmt.Errorf("Invalid duration format %s for option --%s: %s", o.PullRequestPollTime, optionPullRequestPollTime, err)
		}
		o.PullRequestPollDuration = &duration
	}

	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	if o.Namespace == "" {
		o.Namespace = devNs
	}
	ns := o.Namespace

	o.workflowMap = map[string]*v1.Workflow{}
	o.pipelineMap = map[string]*v1.PipelineActivity{}

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

	ticker := time.NewTicker(*o.PullRequestPollDuration)
	go func() {
		for t := range ticker.C {
			if o.Verbose {
				log.Infof("Polling to see if any PRs have merged: %v\n", t)
			}
			//o.pollGitPipelineStatuses(jxClient, ns)
			o.ReloadAndPollGitPipelineStatuses(jxClient, ns)
		}
	}()

	// Wait forever
	select {}
}

func (o *ControllerWorkflowOptions) PipelineMap() map[string]*v1.PipelineActivity {
	return o.pipelineMap
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
		o.pipelineMap[pipeline.Name] = &pipeline
		o.onActivity(&pipeline, jxClient, ns)
	}
	return nil
}

func (o *ControllerWorkflowOptions) onWorkflowObj(obj interface{}, jxClient versioned.Interface, ns string) {
	workflow, ok := obj.(*v1.Workflow)
	if !ok {
		log.Warnf("Object is not a Workflow %#v\n", obj)
		return
	}
	if workflow != nil {
		o.onWorkflow(workflow, jxClient, ns)
	}
}

func (o *ControllerWorkflowOptions) deleteWorkflowObjb(obj interface{}, jxClient versioned.Interface, ns string) {
	workflow, ok := obj.(*v1.Workflow)
	if !ok {
		log.Warnf("Object is not a Workflow %#v\n", obj)
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
		log.Warnf("Object is not a PipelineActivity %#v\n", obj)
		return
	}
	if pipeline != nil {
		activity, err := jxClient.JenkinsV1().PipelineActivities(ns).Get(pipeline.Name, metav1.GetOptions{})
		if err == nil {
			if kube.IsResourceVersionNewer(activity.ResourceVersion, pipeline.ResourceVersion) {
				if o.Verbose {
					log.Infof("onActivity %s using newer resourceVersion of PipelineActivity %s > %s\n", pipeline.Name, activity.ResourceVersion, pipeline.ResourceVersion)
				}
				pipeline = activity
			}
		}
		o.onActivity(pipeline, jxClient, ns)
	}
}

func (o *ControllerWorkflowOptions) onActivity(pipeline *v1.PipelineActivity, jxClient versioned.Interface, ns string) {
	workflowName := pipeline.Spec.Workflow
	version := pipeline.Spec.Version
	repoName := pipeline.RepositoryName()
	branch := pipeline.BranchName()
	pipelineName := pipeline.Spec.Pipeline
	build := pipeline.Spec.Build

	if o.Verbose {
		log.Infof("Processing pipeline %s repo %s version %s with workflow %s and status %s\n", pipeline.Name, repoName, version, workflowName, string(pipeline.Spec.WorkflowStatus))
	}

	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	if repoName == "" || version == "" || build == "" || pipelineName == "" {
		if o.Verbose {
			log.Infof("Ignoring missing data for pipeline: %s repo: %s version: %s status: %s\n", pipeline.Name, repoName, version, string(pipeline.Spec.WorkflowStatus))
		}
		o.removePipelineActivity(pipeline, activities)
		return
	}

	if workflowName == "" {
		o.removePipelineActivityIfNoManual(pipeline, activities)
		return
	}

	if !pipeline.Spec.WorkflowStatus.IsTerminated() {
		flow := o.workflowMap[workflowName]
		if flow == nil && workflowName == "default" {
			var err error
			flow, err = workflow.CreateDefaultWorkflow(jxClient, ns)
			if err != nil {
				log.Warnf("Cannot create default Workflow: %s\n", err)
				o.removePipelineActivity(pipeline, activities)
				return
			}
		}

		if flow == nil {
			o.removePipelineActivityIfNoManual(pipeline, activities)
			return
		}

		if !o.isNewestPipeline(pipeline, activities) {
			return
		}

		if !o.isReleaseBranch(branch) {
			log.Infof("Ignoring branch %s\n", branch)
			o.removePipelineActivity(pipeline, activities)
			return
		}

		// ensure the pipeline is in our map
		o.pipelineMap[pipeline.Name] = pipeline

		// lets walk the Workflow spec and see if we need to trigger any PRs or move the PipelineActivity forward
		promoteStatusMap := createPromoteStatus(pipeline)

		allStepsComplete := true
		for _, step := range flow.Spec.Steps {
			promote := step.Promote
			if promote != nil {
				envName := promote.Environment
				if envName != "" {
					status := promoteStatusMap[envName]
					if status == nil || status.PullRequest == nil || status.PullRequest.PullRequestURL == "" {
						allStepsComplete = false
						// can we generate a PR now?
						if canExecuteStep(flow, pipeline, &step, promoteStatusMap, envName) {
							log.Infof("Creating PR for environment %s from PipelineActivity %s as current status is %#v\n", envName, pipeline.Name, status)
							po := o.createPromoteOptions(repoName, envName, pipelineName, build, version)

							err := po.Run()
							if err != nil {
								log.Warnf("Failed to create PullRequest on pipeline %s repo %s version %s with workflow %s: %s\n", pipeline.Name, repoName, version, workflowName, err)
							}
						}
					}
					if status != nil && status.Status != v1.ActivityStatusTypeSucceeded {
						allStepsComplete = false
					}
				}
			}
		}
		if allStepsComplete && (pipeline.Spec.Status != v1.ActivityStatusTypeSucceeded || pipeline.Spec.WorkflowStatus != v1.ActivityStatusTypeSucceeded) {
			pipeline.Spec.Status = v1.ActivityStatusTypeSucceeded
			pipeline.Spec.WorkflowStatus = v1.ActivityStatusTypeSucceeded
			_, err := jxClient.JenkinsV1().PipelineActivities(ns).Update(pipeline)
			if err != nil {
				log.Warnf("Failed to update PipelineActivity %s due to being complete: %s", pipeline.Name, err)
			}

		}
	}
}

func (o *ControllerWorkflowOptions) createPromoteOptions(repoName string, envName string, pipelineName string, build string, version string) *PromoteOptions {
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
		FakePullRequests:  o.FakePullRequests,
		Namespace:         o.Namespace,
	}
	po.CommonOptions = o.CommonOptions
	po.BatchMode = true
	return po
}

func (o *ControllerWorkflowOptions) createPromoteOptionsFromActivity(pipeline *v1.PipelineActivity, envName string) *PromoteOptions {
	version := pipeline.Spec.Version
	repoName := pipeline.Spec.GitRepository
	pipelineName := pipeline.Spec.Pipeline
	build := pipeline.Spec.Build

	paths := strings.Split(pipelineName, "/")
	if repoName == "" && len(paths) > 1 {
		repoName = paths[len(paths)-2]
	}
	return o.createPromoteOptions(repoName, envName, pipelineName, build, version)
}

func (o *ControllerWorkflowOptions) createGitProviderForPR(prURL string) (gits.GitProvider, *gits.GitRepository, error) {
	// lets remove the id
	idx := strings.LastIndex(prURL, "/")
	if idx <= 0 {
		return nil, nil, fmt.Errorf("No / in URL: %s", prURL)
	}
	gitUrl := prURL[0:idx]
	idx = strings.LastIndex(gitUrl, "/")
	if idx <= 0 {
		return nil, nil, fmt.Errorf("No / in URL: %s", gitUrl)
	}
	gitUrl = gitUrl[0:idx] + ".git"
	if o.FakeGitProvider != nil {
		gitInfo, err := gits.ParseGitURL(gitUrl)
		if err != nil {
			return nil, gitInfo, err
		}
		return o.FakeGitProvider, gitInfo, nil
	}
	answer, gitInfo, err := o.createGitProviderForURLWithoutKind(gitUrl)
	if err != nil {
		return answer, gitInfo, errors.Wrapf(err, "Failed for git URL %s", gitUrl)
	}
	return answer, gitInfo, nil
}

func (o *ControllerWorkflowOptions) createGitProvider(activity *v1.PipelineActivity) (gits.GitProvider, *gits.GitRepository, error) {
	gitUrl := activity.Spec.GitURL
	if gitUrl == "" {
		return nil, nil, fmt.Errorf("No GitURL for PipelineActivity %s", activity.Name)
	}
	if o.FakeGitProvider != nil {
		gitInfo, err := gits.ParseGitURL(gitUrl)
		if err != nil {
			return nil, gitInfo, err
		}
		return o.FakeGitProvider, gitInfo, nil
	}
	answer, gitInfo, err := o.createGitProviderForURLWithoutKind(gitUrl)
	if err != nil {
		return answer, gitInfo, errors.Wrapf(err, "Failed for git URL %s", gitUrl)
	}
	return answer, gitInfo, nil
}

// pollGitPipelineStatuses lets poll all the pending PipelineActivity resources to see if any of them
// have PR has merged or the pipeline on master has completed
func (o *ControllerWorkflowOptions) pollGitPipelineStatuses(jxClient versioned.Interface, ns string) {
	environments := jxClient.JenkinsV1().Environments(ns)
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	for _, activity := range o.pipelineMap {
		o.pollGitStatusforPipeline(activity, activities, environments, ns)
	}
}

// ReloadAndPollGitPipelineStatuses reloads all the current pending PipelineActivity objects and polls their Git
// status to see if the workflows can progress.
//
// Note this method is only really for testing and simulation
func (o *ControllerWorkflowOptions) ReloadAndPollGitPipelineStatuses(jxClient versioned.Interface, ns string) {
	environments := jxClient.JenkinsV1().Environments(ns)
	activities := jxClient.JenkinsV1().PipelineActivities(ns)

	pipelines, err := activities.List(metav1.ListOptions{})
	if err != nil {
		log.Warnf("failed to list PipelineActivity resources: %s", err)
	} else {
		for _, pipeline := range pipelines.Items {
			if o.Verbose {
				log.Infof("Polling git status of activity %s\n", pipeline.Name)
			}
			o.pollGitStatusforPipeline(&pipeline, activities, environments, ns)
		}
	}
}

// pollGitStatusforPipeline polls the pending PipelineActivity resources to see if the
// PR has merged or the pipeline on master has completed
func (o *ControllerWorkflowOptions) pollGitStatusforPipeline(activity *v1.PipelineActivity, activities typev1.PipelineActivityInterface, environments typev1.EnvironmentInterface, ns string) {
	if !o.isReleaseBranch(activity.BranchName()) {
		o.removePipelineActivity(activity, activities)
		return
	}

	// TODO should be is newest pipeline for this environment promote...
	if !o.isNewestPipeline(activity, activities) {
		return
	}

	for _, step := range activity.Spec.Steps {
		promote := step.Promote
		if promote == nil {
			continue
		}
		if promote.Status.IsTerminated() {
			if o.Verbose {
				log.Infof("Pipeline %s promote Environment %s ignored as status %s\n", activity.Name, promote.Environment, string(promote.Status))
			}
			continue
		}
		envName := promote.Environment
		pullRequestStep := promote.PullRequest
		if pullRequestStep == nil {
			log.Infof("Pipeline %s promote Environment %s status %s ignored as no PullRequest\n", activity.Name, promote.Environment, string(promote.Status))
			continue
		}
		prURL := pullRequestStep.PullRequestURL
		if prURL == "" || envName == "" {
			log.Infof("Pipeline %s promote Environment %s status %s ignored for PR %s\n", activity.Name, promote.Environment, string(promote.Status), prURL)
			continue
		}
		gitProvider, gitInfo, err := o.createGitProviderForPR(prURL)
		if err != nil {
			log.Warnf("Failed to create git Provider: %s", err)
			return
		}
		if gitProvider == nil || gitInfo == nil {
			return
		}
		prNumber, err := PullRequestURLToNumber(prURL)
		if err != nil {
			log.Warnf("Failed to get PR number: %s", err)
			return
		}
		pr, err := gitProvider.GetPullRequest(gitInfo.Organisation, gitInfo, prNumber)
		if err != nil {
			log.Warnf("Failed to query the Pull Request status on pipeline %s for repo %s PR %d for PR %s: %s", activity.Name, gitInfo.HttpsURL(), prNumber, prURL, err)
		} else {
			if o.Verbose {
				log.Infof("Pipeline %s promote Environment %s has PR %s\n", activity.Name, envName, prURL)
			}
			po := o.createPromoteOptionsFromActivity(activity, envName)

			if pr.Merged != nil && *pr.Merged {
				if pr.MergeCommitSHA == nil {
					log.Warnf("Pipeline %s promote Environment %s has PR %s which is merged but there is no merge SHA\n", activity.Name, envName, prURL)
				} else {
					mergeSha := *pr.MergeCommitSHA
					mergedPR := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
						kube.CompletePromotionPullRequest(a, s, ps, p)
						p.MergeCommitSHA = mergeSha
						return nil
					}
					env, err := environments.Get(envName, metav1.GetOptions{})
					if err != nil {
						log.Warnf("Failed to find environment %s: %s\n", envName, err)
						return
					} else {
						promoteKey := po.createPromoteKey(env)

						promoteKey.OnPromotePullRequest(o.jxClient, o.Namespace, mergedPR)
						promoteKey.OnPromoteUpdate(o.jxClient, o.Namespace, kube.StartPromotionUpdate)

						statuses, err := gitProvider.ListCommitStatus(pr.Owner, pr.Repo, mergeSha)
						if err == nil {
							urlStatusMap := map[string]string{}
							urlStatusTargetURLMap := map[string]string{}
							if len(statuses) > 0 {
								for _, status := range statuses {
									if status.IsFailed() {
										log.Warnf("merge status: %s URL: %s description: %s\n",
											status.State, status.TargetURL, status.Description)
										return
									}
									url := status.URL
									state := status.State
									if urlStatusMap[url] == "" || urlStatusMap[url] != gitStatusSuccess {
										if urlStatusMap[url] != state {
											urlStatusMap[url] = state
											urlStatusTargetURLMap[url] = status.TargetURL
										}
									}
								}
								prStatuses := []v1.GitStatus{}
								keys := util.SortedMapKeys(urlStatusMap)
								for _, url := range keys {
									state := urlStatusMap[url]
									targetURL := urlStatusTargetURLMap[url]
									if targetURL == "" {
										targetURL = url
									}
									prStatuses = append(prStatuses, v1.GitStatus{
										URL:    targetURL,
										Status: state,
									})
								}
								updateStatuses := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromoteUpdateStep) error {
									p.Statuses = prStatuses
									return nil
								}
								promoteKey.OnPromoteUpdate(o.jxClient, o.Namespace, updateStatuses)

								succeeded := true
								for _, v := range urlStatusMap {
									if v != gitStatusSuccess {
										succeeded = false
									}
								}
								if succeeded {
									gitURL := activity.Spec.GitURL
									if gitURL == "" {
										log.Warnf("No git URL for PipelineActivity %s so cannot comment on issues\n", activity.Name)
										return
									}
									gitInfo, err := gits.ParseGitURL(gitURL)
									if err != nil {
										log.Warnf("Failed to parse Git URL %s for PipelineActivity %s so cannot comment on issues: %s", gitURL, activity.Name, err)
										return
									}
									po.GitInfo = gitInfo
									err = po.commentOnIssues(ns, env, promoteKey)
									if err != nil {
										log.Warnf("Failed to comment on issues: %s", err)
										return
									}
									err = promoteKey.OnPromoteUpdate(o.jxClient, o.Namespace, kube.CompletePromotionUpdate)
									if err != nil {
										log.Warnf("Failed to update PipelineActivity on promotion completion: %s", err)
									}
									return
								}
							}
						}
					}
				}
			} else {
				if pr.IsClosed() {
					log.Warnf("Pull Request %s is closed\n", util.ColorInfo(pr.URL))
					// TODO should we mark the PipelineActivity as complete?
					return
				}

				// lets try merge if the status is good
				status, err := gitProvider.PullRequestLastCommitStatus(pr)
				if err != nil {
					log.Warnf("Failed to query the Pull Request last commit status for %s ref %s %s\n", pr.URL, pr.LastCommitSha, err)
					//return fmt.Errorf("Failed to query the Pull Request last commit status for %s ref %s %s", pr.URL, pr.LastCommitSha, err)
				} else if status == "in-progress" {
					log.Infoln("The build for the Pull Request last commit is currently in progress.")
				} else {
					log.Infof("Pipeline %s promote Environment %s has PR %s with status %s\n", activity.Name, envName, prURL, status)

					if status == "success" {
						if !o.NoMergePullRequest {
							err = gitProvider.MergePullRequest(pr, "jx promote automatically merged promotion PR")
							if err != nil {
								log.Warnf("Failed to merge the Pull Request %s due to %s maybe I don't have karma?\n", pr.URL, err)
							}
						}
					} else if status == "error" || status == "failure" {
						log.Warnf("Pull request %s last commit has status %s for ref %s", pr.URL, status, pr.LastCommitSha)
						return
					}
				}
			}
			if pr.Mergeable != nil && !*pr.Mergeable {
				log.Infoln("Rebasing PullRequest due to conflict")
				env, err := environments.Get(envName, metav1.GetOptions{})
				if err != nil {
					log.Warnf("Failed to find environment %s: %s\n", envName, err)
				} else {
					releaseInfo := o.createReleaseInfo(activity, env)
					if releaseInfo != nil {
						err = po.PromoteViaPullRequest(env, releaseInfo)
					}
				}
			}
		}
	}
}

func (o *ControllerWorkflowOptions) createReleaseInfo(activity *v1.PipelineActivity, env *v1.Environment) *ReleaseInfo {
	spec := &activity.Spec
	app := activity.RepositoryName()
	if app == "" {
		return nil
	}
	fullAppName := app
	if o.LocalHelmRepoName != "" {
		fullAppName = o.LocalHelmRepoName + "/" + app
	}
	releaseName := "" // TODO o.ReleaseName
	if releaseName == "" {
		releaseName = env.Spec.Namespace + "-" + app
		o.ReleaseName = releaseName
	}
	return &ReleaseInfo{
		ReleaseName: releaseName,
		FullAppName: fullAppName,
		Version:     spec.Version,
	}
}

func canExecuteStep(workflow *v1.Workflow, activity *v1.PipelineActivity, step *v1.WorkflowStep, statusMap map[string]*v1.PromoteActivityStep, promoteToEnv string) bool {
	for _, envName := range step.Preconditions.Environments {
		status := statusMap[envName]
		if status == nil {
			log.Warnf("Cannot promote to Environment: %s as precondition Environment: %s as no status\n", promoteToEnv, envName)
			return false
		}
		if status.Status != v1.ActivityStatusTypeSucceeded {
			log.Warnf("Cannot promote to Environment: %s as precondition Environment: %s has status %s", promoteToEnv, envName, string(status.Status))
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

// createPromoteStepActivityKey deduces the pipeline metadata from the Knative workflow pod
func (o *ControllerWorkflowOptions) createPromoteStepActivityKey(buildName string, pod *corev1.Pod) *kube.PromoteStepActivityKey {
	branch := ""
	lastCommitSha := ""
	lastCommitMessage := ""
	lastCommitURL := ""
	build := DigitSuffix(buildName)
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
		log.Warnf("Failed to parse Git URL %s: %s", gitUrl, err)
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

// PullRequestURLToNumber turns Pull Request URL to number
func PullRequestURLToNumber(text string) (int, error) {
	paths := strings.Split(strings.TrimSuffix(text, "/"), "/")
	lastPath := paths[len(paths)-1]
	prNumber, err := strconv.Atoi(lastPath)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to parse PR number from %s on URL %s", lastPath, text)
	}
	return prNumber, nil
}

func (o *ControllerWorkflowOptions) isReleaseBranch(branchName string) bool {
	// TODO look in TeamSettings for a list of allowed release branch patterns
	return branchName == "master"
}

func noopCallback(activity *v1.PipelineActivity) bool {
	return true
}

func setActivitySucceeded(activity *v1.PipelineActivity) bool {
	activity.Spec.Status = v1.ActivityStatusTypeSucceeded
	activity.Spec.WorkflowStatus = v1.ActivityStatusTypeSucceeded
	return true
}

func setActivityAborted(activity *v1.PipelineActivity) bool {
	activity.Spec.Status = v1.ActivityStatusTypeAborted
	activity.Spec.WorkflowStatus = v1.ActivityStatusTypeAborted
	activity.Spec.WorkflowMessage = "Due to newer pipeline"
	return true
}

func (o *ControllerWorkflowOptions) removePipelineActivity(activity *v1.PipelineActivity, activities typev1.PipelineActivityInterface) {
	o.modifyAndRemovePipelineActivity(activity, activities, noopCallback)
}

// removePipelineActivityIfNoManual only remove the PipelineActivity if there is not any pending Promote
func (o *ControllerWorkflowOptions) removePipelineActivityIfNoManual(activity *v1.PipelineActivity, activities typev1.PipelineActivityInterface) {
	for _, step := range activity.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			if promote.Status == v1.ActivityStatusTypePending || promote.Status == v1.ActivityStatusTypeRunning {
				return
			}
		}
	}
	o.removePipelineActivity(activity, activities)
}

func (o *ControllerWorkflowOptions) modifyAndRemovePipelineActivity(activity *v1.PipelineActivity, activities typev1.PipelineActivityInterface, callback func(activity *v1.PipelineActivity) bool) error {
	err := modifyPipeline(activities, activity, callback)
	delete(o.pipelineMap, activity.Name)
	return err
}

func modifyPipeline(activities typev1.PipelineActivityInterface, activity *v1.PipelineActivity, callback func(activity *v1.PipelineActivity) bool) error {
	old := activity
	if callback(activity) {
		if !reflect.DeepEqual(activity, &old) {
			_, err := activities.Update(activity)
			if err != nil {
				log.Warnf("Failed to update PipelineActivity %s: %s\n", activity.Name, err)
				return err
			}
		}
	}
	return nil
}

// isNewestPipeline returns true if this pipeline is the newest pipeline version for a repo
func (o *ControllerWorkflowOptions) isNewestPipeline(activity *v1.PipelineActivity, activities typev1.PipelineActivityInterface) bool {
	newest := true
	deleteNames := []*v1.PipelineActivity{}
	for _, act2 := range o.pipelineMap {
		if activity.Spec.Pipeline == act2.Spec.Pipeline {
			b1 := activity.Spec.Build
			b2 := act2.Spec.Build
			if b1 != b2 {
				if kube.IsResourceVersionNewer(b2, b1) {
					newest = false
				} else if kube.IsResourceVersionNewer(b1, b2) {
					deleteNames = append(deleteNames, act2)
				}
			}
		}
	}
	for _, p := range deleteNames {
		if o.Verbose {
			log.Infof("Removing old Pipeline version %s\n", p.Name)
		}
		o.modifyAndRemovePipelineActivity(p, activities, setActivityAborted)
	}
	if !newest {
		if o.Verbose {
			log.Infof("Removing old Pipeline version %s\n", activity.Name)
		}
		o.modifyAndRemovePipelineActivity(activity, activities, setActivityAborted)
	}
	return newest
}
