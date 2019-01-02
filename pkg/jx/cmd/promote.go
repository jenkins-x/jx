package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/kube/services"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	optionEnvironment         = "env"
	optionApplication         = "app"
	optionTimeout             = "timeout"
	optionPullRequestPollTime = "pull-request-poll-time"

	gitStatusSuccess = "success"
)

var (
	waitAfterPullRequestCreated = time.Second * 3
)

// PromoteOptions containers the CLI options
type PromoteOptions struct {
	CommonOptions

	Namespace               string
	Environment             string
	Application             string
	Pipeline                string
	Build                   string
	Version                 string
	ReleaseName             string
	LocalHelmRepoName       string
	HelmRepositoryURL       string
	NoHelmUpdate            bool
	AllAutomatic            bool
	NoMergePullRequest      bool
	NoPoll                  bool
	NoWaitAfterMerge        bool
	IgnoreLocalFiles        bool
	NoWaitForUpdatePipeline bool
	Timeout                 string
	PullRequestPollTime     string
	Filter                  string
	Alias                   string

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback ConfigureGitFolderFn

	// for testing
	FakePullRequests CreateEnvPullRequestFn
	UseFakeHelm      bool

	// calculated fields
	TimeoutDuration         *time.Duration
	PullRequestPollDuration *time.Duration
	Activities              typev1.PipelineActivityInterface
	GitInfo                 *gits.GitRepository
	jenkinsURL              string
	releaseResource         *v1.Release
	ReleaseInfo             *ReleaseInfo
}

type ReleaseInfo struct {
	ReleaseName     string
	FullAppName     string
	Version         string
	PullRequestInfo *gits.PullRequestInfo
}

var (
	promote_long = templates.LongDesc(`
		Promotes a version of an application to zero to many permanent environments.

		For more documentation see: [https://jenkins-x.io/about/features/#promotion](https://jenkins-x.io/about/features/#promotion)

`)

	promote_example = templates.Examples(`
		# Promote a version of the current application to staging 
        # discovering the application name from the source code
		jx promote --version 1.2.3 --env staging

		# Promote a version of the myapp application to production
		jx promote myapp --version 1.2.3 --env production

		# To search for all the available charts for a given name use -f.
		# e.g. to find a redis chart to install
		jx promote -f redis

		# To promote a postgres chart using an alias
		jx promote -f postgres --alias mydb

		# To create or update a Preview Environment please see the 'jx preview' command
		jx preview
	`)
)

// NewCmdPromote creates the new command for: jx get prompt
func NewCmdPromote(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &PromoteOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "promote [application]",
		Short:   "Promotes a version of an application to an Environment",
		Long:    promote_long,
		Example: promote_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The Namespace to promote to")
	cmd.Flags().StringVarP(&options.Environment, optionEnvironment, "e", "", "The Environment to promote to")
	cmd.Flags().BoolVarP(&options.AllAutomatic, "all-auto", "", false, "Promote to all automatic environments in order")

	options.addPromoteOptions(cmd)
	return cmd
}

func (options *PromoteOptions) addPromoteOptions(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&options.Application, optionApplication, "a", "", "The Application to promote")
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "The search filter to find charts to promote")
	cmd.Flags().StringVarP(&options.Alias, "alias", "", "", "The optional alias used in the 'requirements.yaml' file")
	cmd.Flags().StringVarP(&options.Pipeline, "pipeline", "", "", "The Pipeline string in the form 'folderName/repoName/branch' which is used to update the PipelineActivity. If not specified its defaulted from  the '$BUILD_NUMBER' environment variable")
	cmd.Flags().StringVarP(&options.Build, "build", "", "", "The Build number which is used to update the PipelineActivity. If not specified its defaulted from  the '$BUILD_NUMBER' environment variable")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The Version to promote")
	cmd.Flags().StringVarP(&options.LocalHelmRepoName, "helm-repo-name", "r", kube.LocalHelmRepoName, "The name of the helm repository that contains the app")
	cmd.Flags().StringVarP(&options.HelmRepositoryURL, "helm-repo-url", "u", helm.DefaultHelmRepositoryURL, "The Helm Repository URL to use for the App")
	cmd.Flags().StringVarP(&options.ReleaseName, "release", "", "", "The name of the helm release")
	cmd.Flags().StringVarP(&options.Timeout, optionTimeout, "t", "1h", "The timeout to wait for the promotion to succeed in the underlying Environment. The command fails if the timeout is exceeded or the promotion does not complete")
	cmd.Flags().StringVarP(&options.PullRequestPollTime, optionPullRequestPollTime, "", "20s", "Poll time when waiting for a Pull Request to merge")
	cmd.Flags().BoolVarP(&options.NoHelmUpdate, "no-helm-update", "", false, "Allows the 'helm repo update' command if you are sure your local helm cache is up to date with the version you wish to promote")
	cmd.Flags().BoolVarP(&options.NoMergePullRequest, "no-merge", "", false, "Disables automatic merge of promote Pull Requests")
	cmd.Flags().BoolVarP(&options.NoPoll, "no-poll", "", false, "Disables polling for Pull Request or Pipeline status")
	cmd.Flags().BoolVarP(&options.NoWaitAfterMerge, "no-wait", "", false, "Disables waiting for completing promotion after the Pull request is merged")
	cmd.Flags().BoolVarP(&options.IgnoreLocalFiles, "ignore-local-file", "", false, "Ignores the local file system when deducing the Git repository")
}

// Run implements this command
func (o *PromoteOptions) Run() error {
	app := o.Application
	if app == "" {
		args := o.Args
		if len(args) == 0 {
			search := o.Filter
			var err error
			if search != "" {
				app, err = o.SearchForChart(search)
			} else {
				app, err = o.DiscoverAppName()
			}
			if err != nil {
				return err
			}
		} else {
			app = args[0]
		}
	}
	o.Application = app

	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	prow, err := o.isProw()
	if err != nil {
		return err
	}
	if prow {
		log.Warn("prow based install so skip waiting for the merge of Pull Requests to go green as currently there is an issue with getting" +
			"statuses from the PR, see https://github.com/jenkins-x/jx/issues/2410")
		o.NoWaitForUpdatePipeline = true
	}

	if o.Environment == "" && !o.BatchMode {
		names := []string{}
		m, allEnvNames, err := kube.GetOrderedEnvironments(jxClient, ns)
		if err != nil {
			return err
		}
		for _, n := range allEnvNames {
			env := m[n]
			if env.Spec.Kind == v1.EnvironmentKindTypePermanent {
				names = append(names, n)
			}
		}
		o.Environment, err = kube.PickEnvironment(names, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}

	if o.PullRequestPollTime != "" {
		duration, err := time.ParseDuration(o.PullRequestPollTime)
		if err != nil {
			return fmt.Errorf("Invalid duration format %s for option --%s: %s", o.PullRequestPollTime, optionPullRequestPollTime, err)
		}
		o.PullRequestPollDuration = &duration
	}
	if o.Timeout != "" {
		duration, err := time.ParseDuration(o.Timeout)
		if err != nil {
			return fmt.Errorf("Invalid duration format %s for option --%s: %s", o.Timeout, optionTimeout, err)
		}
		o.TimeoutDuration = &duration
	}

	targetNS, env, err := o.GetTargetNamespace(o.Namespace, o.Environment)
	if err != nil {
		return err
	}

	o.Activities = jxClient.JenkinsV1().PipelineActivities(ns)

	releaseName := o.ReleaseName
	if releaseName == "" {
		releaseName = targetNS + "-" + app
		o.ReleaseName = releaseName
	}

	if o.AllAutomatic {
		return o.PromoteAllAutomatic()
	}
	if env == nil {
		if o.Environment == "" {
			return util.MissingOption(optionEnvironment)
		}
		env, err := jxClient.JenkinsV1().Environments(ns).Get(o.Environment, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if env == nil {
			return fmt.Errorf("Could not find an Environment called %s", o.Environment)
		}
	}
	releaseInfo, err := o.Promote(targetNS, env, true)
	if err != nil {
		return err
	}

	o.ReleaseInfo = releaseInfo
	if !o.NoPoll {
		err = o.WaitForPromotion(targetNS, env, releaseInfo)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *PromoteOptions) PromoteAllAutomatic() error {
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	team, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}
	jxClient, _, err := o.JXClient()
	if err != nil {
		return err
	}
	envs, err := jxClient.JenkinsV1().Environments(team).List(metav1.ListOptions{})
	if err != nil {
		log.Warnf("No Environments found: %s/n", err)
		return nil
	}
	environments := envs.Items
	if len(environments) == 0 {
		log.Warnf("No Environments have been created yet in team %s. Please create some via 'jx create env'\n", team)
		return nil
	}
	kube.SortEnvironments(environments)

	for _, env := range environments {
		kind := env.Spec.Kind
		if env.Spec.PromotionStrategy == v1.PromotionStrategyTypeAutomatic && kind.IsPermanent() {
			ns := env.Spec.Namespace
			if ns == "" {
				return fmt.Errorf("No namespace for environment %s", env.Name)
			}
			releaseInfo, err := o.Promote(ns, &env, false)
			if err != nil {
				return err
			}
			o.ReleaseInfo = releaseInfo
			if !o.NoPoll {
				err = o.WaitForPromotion(ns, &env, releaseInfo)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (o *PromoteOptions) Promote(targetNS string, env *v1.Environment, warnIfAuto bool) (*ReleaseInfo, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	app := o.Application
	if app == "" {
		log.Warnf("No application name could be detected so cannot promote via Helm. If the detection of the helm chart name is not working consider adding it with the --%s argument on the 'jx promomote' command\n", optionApplication)
		return nil, nil
	}
	version := o.Version
	info := util.ColorInfo
	if version == "" {
		log.Infof("Promoting latest version of app %s to namespace %s\n", info(app), info(targetNS))
	} else {
		log.Infof("Promoting app %s version %s to namespace %s\n", info(app), info(version), info(targetNS))
	}
	fullAppName := app
	if o.LocalHelmRepoName != "" {
		fullAppName = o.LocalHelmRepoName + "/" + app
	}
	releaseName := o.ReleaseName
	if releaseName == "" {
		releaseName = targetNS + "-" + app
		o.ReleaseName = releaseName
	}
	releaseInfo := &ReleaseInfo{
		ReleaseName: releaseName,
		FullAppName: fullAppName,
		Version:     version,
	}

	if warnIfAuto && env != nil && env.Spec.PromotionStrategy == v1.PromotionStrategyTypeAutomatic && !o.BatchMode {
		log.Infof("%s", util.ColorWarning(fmt.Sprintf("WARNING: The Environment %s is setup to promote automatically as part of the CI/CD Pipelines.\n\n", env.Name)))

		confirm := &survey.Confirm{
			Message: "Do you wish to promote anyway? :",
			Default: false,
		}
		flag := false
		err := survey.AskOne(confirm, &flag, nil, surveyOpts)
		if err != nil {
			return releaseInfo, err
		}
		if !flag {
			return releaseInfo, nil
		}
	}

	promoteKey := o.createPromoteKey(env)
	if env != nil {
		source := &env.Spec.Source
		if source.URL != "" && env.Spec.Kind.IsPermanent() {
			err := o.PromoteViaPullRequest(env, releaseInfo)
			if err == nil {
				startPromotePR := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
					kube.StartPromotionPullRequest(a, s, ps, p)
					pr := releaseInfo.PullRequestInfo
					if pr != nil && pr.PullRequest != nil && p.PullRequestURL == "" {
						p.PullRequestURL = pr.PullRequest.URL
					}
					if version != "" && a.Spec.Version == "" {
						a.Spec.Version = version
					}
					return nil
				}
				err = promoteKey.OnPromotePullRequest(o.Activities, startPromotePR)
				if err != nil {
					log.Warnf("Failed to update PipelineActivity: %s\n", err)
				}
				// lets sleep a little before we try poll for the PR status
				time.Sleep(waitAfterPullRequestCreated)
			}
			return releaseInfo, err
		}
	}

	var err error
	if !o.UseFakeHelm {
		err := o.verifyHelmConfigured()
		if err != nil {
			return releaseInfo, err
		}
	}

	// lets do a helm update to ensure we can find the latest version
	if !o.NoHelmUpdate {
		log.Info("Updating the helm repositories to ensure we can find the latest versions...")
		err = o.Helm().UpdateRepo()
		if err != nil {
			return releaseInfo, err
		}
	}

	startPromote := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromoteUpdateStep) error {
		kube.StartPromotionUpdate(a, s, ps, p)
		if version != "" && a.Spec.Version == "" {
			a.Spec.Version = version
		}
		return nil
	}
	promoteKey.OnPromoteUpdate(o.Activities, startPromote)

	err = o.Helm().UpgradeChart(fullAppName, releaseName, targetNS, &version, true, nil, false, true, nil, nil, "",
		"", "")
	if err == nil {
		err = o.commentOnIssues(targetNS, env, promoteKey)
		if err != nil {
			log.Warnf("Failed to comment on issues for release %s: %s\n", releaseName, err)
		}
		err = promoteKey.OnPromoteUpdate(o.Activities, kube.CompletePromotionUpdate)
	} else {
		err = promoteKey.OnPromoteUpdate(o.Activities, kube.FailedPromotionUpdate)
	}
	return releaseInfo, err
}

func (o *PromoteOptions) PromoteViaPullRequest(env *v1.Environment, releaseInfo *ReleaseInfo) error {
	version := o.Version
	versionName := version
	if versionName == "" {
		versionName = "latest"
	}
	app := o.Application

	branchNameText := "promote-" + app + "-" + versionName

	title := app + " to " + versionName
	message := fmt.Sprintf("Promote %s to version %s", app, versionName)

	modifyRequirementsFn := func(requirements *helm.Requirements) error {
		var err error
		if version == "" {
			version, err = o.findLatestVersion(app)
			if err != nil {
				return err
			}
		}
		requirements.SetAppVersion(app, version, o.HelmRepositoryURL, o.Alias)
		return nil
	}
	if o.FakePullRequests != nil {
		info, err := o.FakePullRequests(env, modifyRequirementsFn, branchNameText, title, message, releaseInfo.PullRequestInfo)
		releaseInfo.PullRequestInfo = info
		return err
	} else {
		info, err := o.createEnvironmentPullRequest(env, modifyRequirementsFn, &branchNameText, &title, &message,
			releaseInfo.PullRequestInfo, o.ConfigureGitCallback)
		releaseInfo.PullRequestInfo = info
		return err
	}
}

func (o *PromoteOptions) GetTargetNamespace(ns string, env string) (string, *v1.Environment, error) {
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return "", nil, err
	}
	team, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return "", nil, err
	}

	jxClient, _, err := o.JXClient()
	if err != nil {
		return "", nil, err
	}

	m, envNames, err := kube.GetEnvironments(jxClient, team)
	if err != nil {
		return "", nil, err
	}
	if len(envNames) == 0 {
		return "", nil, fmt.Errorf("No Environments have been created yet in team %s. Please create some via 'jx create env'", team)
	}

	var envResource *v1.Environment
	targetNS := currentNs
	if env != "" {
		envResource = m[env]
		if envResource == nil {
			return "", nil, util.InvalidOption(optionEnvironment, env, envNames)
		}
		targetNS = envResource.Spec.Namespace
		if targetNS == "" {
			return "", nil, fmt.Errorf("Environment %s does not have a namspace associated with it!", env)
		}
	} else if ns != "" {
		targetNS = ns
	}

	labels := map[string]string{}
	annotations := map[string]string{}
	err = kube.EnsureNamespaceCreated(kubeClient, targetNS, labels, annotations)
	if err != nil {
		return "", nil, err
	}
	return targetNS, envResource, nil
}

func (o *PromoteOptions) WaitForPromotion(ns string, env *v1.Environment, releaseInfo *ReleaseInfo) error {
	if o.TimeoutDuration == nil {
		log.Infof("No --%s option specified on the 'jx promote' command so not waiting for the promotion to succeed\n", optionTimeout)
		return nil
	}
	if o.PullRequestPollDuration == nil {
		log.Infof("No --%s option specified on the 'jx promote' command so not waiting for the promotion to succeed\n", optionPullRequestPollTime)
		return nil
	}
	duration := *o.TimeoutDuration
	end := time.Now().Add(duration)

	pullRequestInfo := releaseInfo.PullRequestInfo
	if pullRequestInfo != nil {
		promoteKey := o.createPromoteKey(env)

		err := o.waitForGitOpsPullRequest(ns, env, releaseInfo, end, duration, promoteKey)
		if err != nil {
			// TODO based on if the PR completed or not fail the PR or the Promote?
			promoteKey.OnPromotePullRequest(o.Activities, kube.FailedPromotionPullRequest)
			return err
		}
	}
	return nil
}

// TODO This could do with a refactor and some tests...
func (o *PromoteOptions) waitForGitOpsPullRequest(ns string, env *v1.Environment, releaseInfo *ReleaseInfo, end time.Time, duration time.Duration, promoteKey *kube.PromoteStepActivityKey) error {
	pullRequestInfo := releaseInfo.PullRequestInfo
	logMergeFailure := false
	logNoMergeCommitSha := false
	logHasMergeSha := false
	logMergeStatusError := false
	logNoMergeStatuses := false
	urlStatusMap := map[string]string{}
	urlStatusTargetURLMap := map[string]string{}

	if pullRequestInfo != nil {
		for {
			pr := pullRequestInfo.PullRequest
			gitProvider := pullRequestInfo.GitProvider
			err := gitProvider.UpdatePullRequestStatus(pr)
			if err != nil {
				log.Warnf("Failed to query the Pull Request status for %s %s", pr.URL, err)
			} else {
				if pr.Merged != nil && *pr.Merged {
					if pr.MergeCommitSHA == nil {
						if !logNoMergeCommitSha {
							logNoMergeCommitSha = true
							log.Infof("Pull Request %s is merged but waiting for Merge SHA\n", util.ColorInfo(pr.URL))
						}
					} else {
						mergeSha := *pr.MergeCommitSHA
						if !logHasMergeSha {
							logHasMergeSha = true
							log.Infof("Pull Request %s is merged at sha %s\n", util.ColorInfo(pr.URL), util.ColorInfo(mergeSha))

							mergedPR := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
								kube.CompletePromotionPullRequest(a, s, ps, p)
								p.MergeCommitSHA = mergeSha
								return nil
							}
							promoteKey.OnPromotePullRequest(o.Activities, mergedPR)

							if o.NoWaitAfterMerge {
								log.Infof("Pull requests are merged, No wait on promotion to complete")
								return err
							}
						}

						promoteKey.OnPromoteUpdate(o.Activities, kube.StartPromotionUpdate)

						if o.NoWaitForUpdatePipeline {
							log.Infoln("Pull Request merged but we are not waiting for the update pipeline to complete!")
							err = o.commentOnIssues(ns, env, promoteKey)
							if err == nil {
								err = promoteKey.OnPromoteUpdate(o.Activities, kube.CompletePromotionUpdate)
							}
							return err
						}

						statuses, err := gitProvider.ListCommitStatus(pr.Owner, pr.Repo, mergeSha)
						if err != nil {
							if !logMergeStatusError {
								logMergeStatusError = true
								log.Warnf("Failed to query merge status of repo %s/%s with merge sha %s due to: %s\n", pr.Owner, pr.Repo, mergeSha, err)
							}
						} else {
							if len(statuses) == 0 {
								if !logNoMergeStatuses {
									logNoMergeStatuses = true
									log.Infof("Merge commit has not yet any statuses on repo %s/%s merge sha %s\n", pr.Owner, pr.Repo, mergeSha)
								}
							} else {
								for _, status := range statuses {
									if status.IsFailed() {
										log.Warnf("merge status: %s URL: %s description: %s\n",
											status.State, status.TargetURL, status.Description)
										return fmt.Errorf("Status: %s URL: %s description: %s\n",
											status.State, status.TargetURL, status.Description)
									}
									url := status.URL
									state := status.State
									if urlStatusMap[url] == "" || urlStatusMap[url] != gitStatusSuccess {
										if urlStatusMap[url] != state {
											urlStatusMap[url] = state
											urlStatusTargetURLMap[url] = status.TargetURL
											log.Infof("merge status: %s for URL %s with target: %s description: %s\n",
												util.ColorInfo(state), util.ColorInfo(status.URL), util.ColorInfo(status.TargetURL), util.ColorInfo(status.Description))
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
								promoteKey.OnPromoteUpdate(o.Activities, updateStatuses)

								succeeded := true
								for _, v := range urlStatusMap {
									if v != gitStatusSuccess {
										succeeded = false
									}
								}
								if succeeded {
									log.Infoln("Merge status checks all passed so the promotion worked!")
									err = o.commentOnIssues(ns, env, promoteKey)
									if err == nil {
										err = promoteKey.OnPromoteUpdate(o.Activities, kube.CompletePromotionUpdate)
									}
									return err
								}
							}
						}
					}
				} else {
					if pr.IsClosed() {
						log.Warnf("Pull Request %s is closed\n", util.ColorInfo(pr.URL))
						return fmt.Errorf("Promotion failed as Pull Request %s is closed without merging", pr.URL)
					}

					// lets try merge if the status is good
					status, err := gitProvider.PullRequestLastCommitStatus(pr)
					if err != nil {
						log.Warnf("Failed to query the Pull Request last commit status for %s ref %s %s\n", pr.URL, pr.LastCommitSha, err)
						//return fmt.Errorf("Failed to query the Pull Request last commit status for %s ref %s %s", pr.URL, pr.LastCommitSha, err)
					} else if status == "in-progress" {
						log.Infoln("The build for the Pull Request last commit is currently in progress.")
					} else {
						if status == "success" {
							if !o.NoMergePullRequest {
								err = gitProvider.MergePullRequest(pr, "jx promote automatically merged promotion PR")
								if err != nil {
									if !logMergeFailure {
										logMergeFailure = true
										log.Warnf("Failed to merge the Pull Request %s due to %s maybe I don't have karma?\n", pr.URL, err)
									}
								}
							}
						} else if status == "error" || status == "failure" {
							return fmt.Errorf("Pull request %s last commit has status %s for ref %s", pr.URL, status, pr.LastCommitSha)
						}
					}
				}
				if pr.Mergeable != nil && !*pr.Mergeable {
					log.Infoln("Rebasing PullRequest due to conflict")

					err = o.PromoteViaPullRequest(env, releaseInfo)
					if releaseInfo.PullRequestInfo != nil {
						pullRequestInfo = releaseInfo.PullRequestInfo
					}
				}
			}
			if time.Now().After(end) {
				return fmt.Errorf("Timed out waiting for pull request %s to merge. Waited %s", pr.URL, duration.String())
			}
			time.Sleep(*o.PullRequestPollDuration)
		}
	}
	return nil
}

func (o *PromoteOptions) findLatestVersion(app string) (string, error) {
	versions, err := o.Helm().SearchChartVersions(app)
	if err != nil {
		return "", err
	}

	var maxSemVer *semver.Version
	maxString := ""
	for _, version := range versions {
		sv, err := semver.Parse(version)
		if err != nil {
			log.Warnf("Invalid semantic version: %s %s\n", version, err)
			if maxString == "" || strings.Compare(version, maxString) > 0 {
				maxString = version
			}
		} else {
			if maxSemVer == nil || maxSemVer.Compare(sv) > 0 {
				maxSemVer = &sv
			}
		}
	}

	if maxSemVer != nil {
		return maxSemVer.String(), nil
	}
	if maxString == "" {
		return "", fmt.Errorf("Could not find a version of app %s in the helm repositories", app)
	}
	return maxString, nil
}

func (o *PromoteOptions) verifyHelmConfigured() error {
	helmHomeDir := filepath.Join(util.HomeDir(), ".helm")
	exists, err := util.FileExists(helmHomeDir)
	if err != nil {
		return err
	}
	if !exists {
		log.Warnf("No helm home dir at %s so lets initialise helm client\n", helmHomeDir)

		err = o.helmInit("")
		if err != nil {
			return err
		}
	}

	_, ns, _ := o.KubeClient()
	if err != nil {
		return err
	}

	// lets add the releases chart
	return o.registerLocalHelmRepo(o.LocalHelmRepoName, ns)
}

func (o *PromoteOptions) createPromoteKey(env *v1.Environment) *kube.PromoteStepActivityKey {
	pipeline := o.Pipeline
	build := o.Build
	buildURL := os.Getenv("BUILD_URL")
	buildLogsURL := os.Getenv("BUILD_LOG_URL")
	releaseNotesURL := ""
	gitInfo := o.GitInfo
	if !o.IgnoreLocalFiles {
		var err error
		gitInfo, err = o.Git().Info("")
		releaseName := o.ReleaseName
		if o.releaseResource == nil && releaseName != "" {
			jxClient, _, err := o.JXClient()
			if err == nil && jxClient != nil {
				release, err := jxClient.JenkinsV1().Releases(env.Spec.Namespace).Get(releaseName, metav1.GetOptions{})
				if err == nil && release != nil {
					o.releaseResource = release
				}
			}
		}
		if o.releaseResource != nil {
			releaseNotesURL = o.releaseResource.Spec.ReleaseNotesURL
		}
		if err != nil {
			log.Warnf("Could not discover the Git repository info %s\n", err)
		} else {
			o.GitInfo = gitInfo
		}
	}
	if pipeline == "" {
		pipeline, build = o.getPipelineName(gitInfo, pipeline, build, o.Application)
	}
	if pipeline != "" && build == "" {
		log.Warnf("No $BUILD_NUMBER environment variable found so cannot record promotion activities into the PipelineActivity resources in kubernetes\n")
		var err error
		build, err = o.getLatestPipelineBuildByCRD(pipeline)
		if err != nil {
			log.Warnf("Could not discover the latest PipelineActivity build %s\n", err)
		}
	}
	name := pipeline
	if build != "" {
		name += "-" + build
		if buildURL == "" || buildLogsURL == "" {
			jenkinsURL := o.getJenkinsURL()
			if jenkinsURL != "" {
				path := pipeline
				if !strings.HasPrefix(path, "job/") && !strings.HasPrefix(path, "/job/") {
					// lets split the path and prefix it with /job
					path = strings.Join(strings.Split(path, "/"), "/job/")
					path = util.UrlJoin("job", path)
				}
				path = util.UrlJoin(path, build)
				if !strings.HasSuffix(path, "/") {
					path += "/"
				}
				if buildURL == "" {
					buildURL = util.UrlJoin(jenkinsURL, path)
				}
				if buildLogsURL == "" {
					buildLogsURL = util.UrlJoin(buildURL, "console")
				}
			}
		}
	}
	name = kube.ToValidName(name)
	if o.Verbose {
		log.Infof("Using pipeline: %s build: %s\n", util.ColorInfo(pipeline), util.ColorInfo("#"+build))
	}
	return &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:            name,
			Pipeline:        pipeline,
			Build:           build,
			BuildURL:        buildURL,
			BuildLogsURL:    buildLogsURL,
			GitInfo:         gitInfo,
			ReleaseNotesURL: releaseNotesURL,
		},
		Environment: env.Name,
	}
}

// getLatestPipelineBuild returns the latest pipeline build
func (o *CommonOptions) getLatestPipelineBuildByCRD(pipeline string) (string, error) {
	// lets find the latest build number
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return "", err
	}
	pipelines, err := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	buildNumber := 0
	for _, p := range pipelines.Items {
		if p.Spec.Pipeline == pipeline {
			b := p.Spec.Build
			if b != "" {
				n, err := strconv.Atoi(b)
				if err == nil {
					if n > buildNumber {
						buildNumber = n
					}
				}
			}
		}
	}
	if buildNumber > 0 {
		return strconv.Itoa(buildNumber), nil
	}
	return "1", nil
}

func (o *CommonOptions) getPipelineName(gitInfo *gits.GitRepository, pipeline string, build string, appName string) (string, string) {
	if pipeline == "" {
		pipeline = o.getJobName()
	}
	if build == "" {
		build = o.getBuildNumber()
	}
	if gitInfo != nil && pipeline == "" {
		// lets default the pipeline name from the Git repo
		branch, err := o.Git().Branch(".")
		if err != nil {
			log.Warnf("Could not find the branch name: %s\n", err)
		}
		if branch == "" {
			branch = "master"
		}
		pipeline = util.UrlJoin(gitInfo.Organisation, gitInfo.Name, branch)
	}
	if pipeline == "" && appName != "" {
		suffix := appName + "/master"

		// lets try deduce the pipeline name via the app name
		jxClient, ns, err := o.JXClientAndDevNamespace()
		if err == nil {
			pipelineList, err := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
			if err == nil {
				for _, pipelineResource := range pipelineList.Items {
					pipelineName := pipelineResource.Spec.Pipeline
					if strings.HasSuffix(pipelineName, suffix) {
						pipeline = pipelineName
						break
					}
				}
			}
		}
	}
	if pipeline == "" {
		// lets try find
		log.Warnf("No $JOB_NAME environment variable found so cannot record promotion activities into the PipelineActivity resources in kubernetes\n")
	} else if build == "" {
		// lets validate and determine the current active pipeline branch
		p, b, err := o.getLatestPipelineBuild(pipeline)
		if err != nil {
			log.Warnf("Failed to try detect the current Jenkins pipeline for %s due to %s\n", pipeline, err)
			build = "1"
		} else {
			pipeline = p
			build = b
		}
	}
	return pipeline, build
}

// getLatestPipelineBuild for the given pipeline name lets try find the Jenkins Pipeline and the latest build
func (o *CommonOptions) getLatestPipelineBuild(pipeline string) (string, string, error) {
	log.Infof("pipeline %s\n", pipeline)
	build := ""
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return pipeline, build, err
	}
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return pipeline, build, err
	}
	devEnv, err := kube.GetEnrichedDevEnvironment(kubeClient, jxClient, ns)
	webhookEngine := devEnv.Spec.WebHookEngine
	if webhookEngine == v1.WebHookEngineProw {
		return pipeline, build, nil
	}

	jenkins, err := o.JenkinsClient()
	if err != nil {
		return pipeline, build, err
	}
	paths := strings.Split(pipeline, "/")
	job, err := jenkins.GetJobByPath(paths...)
	if err != nil {
		return pipeline, build, err
	}
	build = strconv.Itoa(job.LastBuild.Number)
	return pipeline, build, nil
}

func (o *PromoteOptions) getJenkinsURL() string {
	if o.jenkinsURL == "" {
		o.jenkinsURL = os.Getenv("JENKINS_URL")
	}
	if o.jenkinsURL == "" {
		o.jenkinsURL = os.Getenv("JENKINS_URL")
	}
	url, err := o.GetJenkinsURL()
	if err != nil {
		log.Warnf("Could not find Jenkins URL: %s", err)
	} else {
		o.jenkinsURL = url
	}
	return o.jenkinsURL
}

// commentOnIssues comments on any issues for a release that the fix is available in the given environment
func (o *PromoteOptions) commentOnIssues(targetNS string, environment *v1.Environment, promoteKey *kube.PromoteStepActivityKey) error {
	ens := environment.Spec.Namespace
	envName := environment.Spec.Label
	app := o.Application
	version := o.Version
	if ens == "" {
		log.Warnf("Environment %s has no namespace\n", envName)
		return nil
	}
	if app == "" {
		log.Warnf("No application name so cannot comment on issues that they are now in %s\n", envName)
		return nil
	}
	if version == "" {
		log.Warnf("No version name so cannot comment on issues that they are now in %s\n", envName)
		return nil
	}
	gitInfo := o.GitInfo
	if gitInfo == nil {
		log.Warnf("No GitInfo discovered so cannot comment on issues that they are now in %s\n", envName)
		return nil
	}
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return err
	}

	provider, err := gitInfo.PickOrCreateProvider(authConfigSvc, "user name to comment on issues", o.BatchMode, gitKind, o.Git(), o.In, o.Out, o.Err)
	if err != nil {
		return err
	}

	releaseName := kube.ToValidNameWithDots(app + "-" + version)
	jxClient, _, err := o.JXClient()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	appNames := []string{app, o.ReleaseName, ens + "-" + app}
	url := ""
	for _, n := range appNames {
		url, err = services.FindServiceURL(kubeClient, ens, n)
		if url != "" {
			break
		}
	}
	if url == "" {
		log.Warnf("Could not find the service URL in namespace %s for names %s\n", ens, strings.Join(appNames, ", "))
	}
	available := ""
	if url != "" {
		available = fmt.Sprintf(" and available [here](%s)", url)
	}

	if available == "" {
		ing, err := kubeClient.ExtensionsV1beta1().Ingresses(ens).Get(app, metav1.GetOptions{})
		if err != nil || ing == nil && o.ReleaseName != "" && o.ReleaseName != app {
			ing, err = kubeClient.ExtensionsV1beta1().Ingresses(ens).Get(o.ReleaseName, metav1.GetOptions{})
		}
		if ing != nil {
			if len(ing.Spec.Rules) > 0 {
				hostname := ing.Spec.Rules[0].Host
				if hostname != "" {
					available = fmt.Sprintf(" and available at %s", hostname)
					url = hostname
				}
			}
		}
	}

	// lets try update the PipelineActivity
	if url != "" && promoteKey.ApplicationURL == "" {
		promoteKey.ApplicationURL = url
		if o.Verbose {
			log.Infof("Application is available at: %s\n", util.ColorInfo(url))
		}
	}

	release, err := jxClient.JenkinsV1().Releases(ens).Get(releaseName, metav1.GetOptions{})
	if err == nil && release != nil {
		o.releaseResource = release
		issues := release.Spec.Issues

		versionMessage := version
		if release.Spec.ReleaseNotesURL != "" {
			versionMessage = "[" + version + "](" + release.Spec.ReleaseNotesURL + ")"
		}
		for _, issue := range issues {
			if issue.IsClosed() {
				log.Infof("Commenting that issue %s is now in %s\n", util.ColorInfo(issue.URL), util.ColorInfo(envName))

				comment := fmt.Sprintf(":white_check_mark: the fix for this issue is now deployed to **%s** in version %s %s", envName, versionMessage, available)
				id := issue.ID
				if id != "" {
					number, err := strconv.Atoi(id)
					if err != nil {
						log.Warnf("Could not parse issue id %s for URL %s\n", id, issue.URL)
					} else {
						if number > 0 {
							err = provider.CreateIssueComment(gitInfo.Organisation, gitInfo.Name, number, comment)
							if err != nil {
								log.Warnf("Failed to add comment to issue %s: %s", issue.URL, err)
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (o *PromoteOptions) SearchForChart(filter string) (string, error) {
	answer := ""
	charts, err := o.Helm().SearchCharts(filter)
	if err != nil {
		return answer, err
	}
	if len(charts) == 0 {
		return answer, fmt.Errorf("No charts available for search filter: %s", filter)
	}
	m := map[string]*helm.ChartSummary{}
	names := []string{}
	for i, chart := range charts {
		text := chart.Name
		if chart.Description != "" {
			text = fmt.Sprintf("%-36s: %s", chart.Name, chart.Description)
		}
		names = append(names, text)
		m[text] = &charts[i]
	}
	name, err := util.PickName(names, "Pick chart to promote: ", "", o.In, o.Out, o.Err)
	if err != nil {
		return answer, err
	}
	chart := m[name]
	chartName := chart.Name
	// TODO now we split the chart into name and repo
	parts := strings.Split(chartName, "/")
	if len(parts) != 2 {
		return answer, fmt.Errorf("Invalid chart name '%s' was expecting single / character separating repo name and chart name", chartName)
	}
	repoName := parts[0]
	appName := parts[1]

	repos, err := o.Helm().ListRepos()
	if err != nil {
		return answer, err
	}

	repoUrl := repos[repoName]
	if repoUrl == "" {
		return answer, fmt.Errorf("Failed to find helm chart repo URL for '%s' when possible values are %s", repoName, util.SortedMapKeys(repos))

	}
	o.Version = chart.ChartVersion
	o.HelmRepositoryURL = repoUrl
	return appName, nil
}
