package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	typev1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"strconv"
)

const (
	optionEnvironment         = "environment"
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

	Namespace           string
	Environment         string
	Application         string
	Version             string
	ReleaseName         string
	LocalHelmRepoName   string
	HelmRepositoryURL   string
	NoHelmUpdate        bool
	AllAutomatic        bool
	Timeout             string
	PullRequestPollTime string

	// calculated fields
	TimeoutDuration         *time.Duration
	PullRequestPollDuration *time.Duration
	Activities              typev1.PipelineActivityInterface
	GitInfo                 *gits.GitRepositoryInfo
	jenkinsURL              string
}

type ReleaseInfo struct {
	ReleaseName     string
	FullAppName     string
	Version         string
	PullRequestInfo *ReleasePullRequestInfo
}

type ReleasePullRequestInfo struct {
	GitProvider          gits.GitProvider
	PullRequest          *gits.GitPullRequest
	PullRequestArguments *gits.GitPullRequestArguments
}

var (
	promote_long = templates.LongDesc(`
		Promotes a version of an application to zero to many permanent environments.
`)

	promote_example = templates.Examples(`
		# Promote a version of the current application to staging 
        # discovering the application name from the source code
		jx promote --version 1.2.3 --env staging

		# Promote a version of the myapp application to production
		jx promote myapp --version 1.2.3 --env prod

		# To create or update a Preview Environment please see the 'jx preview' command
		jx preview
	`)
)

// NewCmdPromote creates the new command for: jx get prompt
func NewCmdPromote(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &PromoteOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "promote [application]",
		Short:   "Promotes a version of an application to an environment",
		Long:    promote_long,
		Example: promote_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
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
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The Version to promote")
	cmd.Flags().StringVarP(&options.LocalHelmRepoName, "helm-repo-name", "r", kube.LocalHelmRepoName, "The name of the helm repository that contains the app")
	cmd.Flags().StringVarP(&options.HelmRepositoryURL, "helm-repo-url", "u", helm.DefaultHelmRepositoryURL, "The Helm Repository URL to use for the App")
	cmd.Flags().StringVarP(&options.ReleaseName, "release", "", "", "The name of the helm release")
	cmd.Flags().StringVarP(&options.Timeout, optionTimeout, "t", "", "The timeout to wait for the promotion to succeed in the underlying Environment. The command fails if the timeout is exceeded or the promotion does not complete")
	cmd.Flags().StringVarP(&options.PullRequestPollTime, optionPullRequestPollTime, "", "20s", "Poll time when waiting for a Pull Request to merge")
	cmd.Flags().BoolVarP(&options.NoHelmUpdate, "no-helm-update", "", false, "Allows the 'helm repo update' command if you are sure your local helm cache is up to date with the version you wish to promote")
}

// Run implements this command
func (o *PromoteOptions) Run() error {
	app := o.Application
	if app == "" {
		args := o.Args
		if len(args) == 0 {
			var err error
			app, err = o.DiscoverAppName()
			if err != nil {
				return err
			}
		} else {
			app = args[0]
		}
	}
	o.Application = app

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

	if o.AllAutomatic {
		return o.PromoteAllAutomatic()
	}
	targetNS, env, err := o.GetTargetNamespace(o.Namespace, o.Environment)
	if err != nil {
		return err
	}
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	jxClient, ns, err := o.JXClient()
	if err != nil {
		return err
	}
	o.Activities = jxClient.JenkinsV1().PipelineActivities(ns)

	releaseInfo, err := o.Promote(targetNS, env, true)
	err = o.WaitForPromotion(targetNS, env, releaseInfo)
	if err != nil {
		return err
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
		o.warnf("No Environments found: %s/n", err)
		return nil
	}
	environments := envs.Items
	if len(environments) == 0 {
		o.warnf("No Environments have been created yet in team %s. Please create some via 'jx create env'\n", team)
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
			err = o.WaitForPromotion(ns, &env, releaseInfo)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *PromoteOptions) Promote(targetNS string, env *v1.Environment, warnIfAuto bool) (*ReleaseInfo, error) {
	app := o.Application
	if app == "" {
		o.warnf("No application name could be detected so cannot promote via Helm. If the detection of the helm chart name is not working consider adding it with the --%s argument on the 'jx promomote' command\n", optionApplication)
		return nil, nil
	}
	version := o.Version
	info := util.ColorInfo
	if version == "" {
		o.Printf("Promoting latest version of app %s to namespace %s\n", info(app), info(targetNS))
	} else {
		o.Printf("Promoting app %s version %s to namespace %s\n", info(app), info(version), info(targetNS))
	}

	fullAppName := app
	if o.LocalHelmRepoName != "" {
		fullAppName = o.LocalHelmRepoName + "/" + app
	}
	releaseName := o.ReleaseName
	if releaseName == "" {
		releaseName = targetNS + "-" + app
	}
	releaseInfo := &ReleaseInfo{
		ReleaseName: releaseName,
		FullAppName: fullAppName,
		Version:     version,
	}

	if warnIfAuto && env != nil && env.Spec.PromotionStrategy == v1.PromotionStrategyTypeAutomatic {
		o.Printf("%s", util.ColorWarning("WARNING: The Environment %s is setup to promote automatically as part of the CI / CD Pipelines.\n\n", env.Name))

		confirm := &survey.Confirm{
			Message: "Do you wish to promote anyway? :",
			Default: false,
		}
		flag := false
		err := survey.AskOne(confirm, &flag, nil)
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
				// lets sleep a little before we try poll for the PR status
				time.Sleep(waitAfterPullRequestCreated)
			}

			return releaseInfo, err
		}
	}

	err := o.verifyHelmConfigured()
	if err != nil {
		return releaseInfo, err
	}

	// lets do a helm update to ensure we can find the latest version
	if !o.NoHelmUpdate {
		o.Printf("Updating the helm repositories to ensure we can find the latest versions...")
		err = o.runCommand("helm", "repo", "update")
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

	if version != "" {
		err = o.runCommand("helm", "upgrade", "--install", "--wait", "--namespace", targetNS, "--version", version, releaseName, fullAppName)
	} else {
		err = o.runCommand("helm", "upgrade", "--install", "--wait", "--namespace", targetNS, releaseName, fullAppName)
	}
	if err == nil {
		err = promoteKey.OnPromoteUpdate(o.Activities, kube.CompletePromotionUpdate)
		if err == nil {
			err = o.commentOnIssues(targetNS, env)
		}
	} else {
		err = promoteKey.OnPromoteUpdate(o.Activities, kube.FailedPromotionUpdate)
	}
	return releaseInfo, err
}

func (o *PromoteOptions) PromoteViaPullRequest(env *v1.Environment, releaseInfo *ReleaseInfo) error {
	source := &env.Spec.Source
	gitURL := source.URL
	if gitURL == "" {
		return fmt.Errorf("No source git URL")
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return err
	}

	environmentsDir, err := util.EnvironmentsDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(environmentsDir, gitInfo.Organisation, gitInfo.Name)

	// now lets clone the fork and push it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return err
	}
	app := o.Application
	version := o.Version
	versionName := version
	if versionName == "" {
		versionName = "latest"
	}

	branchName := gits.ConvertToValidBranchName("promote-" + app + "-" + versionName)
	base := source.Ref
	if base == "" {
		base = "master"
	}

	if exists {
		// lets check the git remote URL is setup correctly
		err = gits.SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "stash")
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "checkout", base)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "pull")
		if err != nil {
			return err
		}
	} else {
		err := os.MkdirAll(dir, DefaultWritePermissions)
		if err != nil {
			return fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		err = gits.GitClone(gitURL, dir)
		if err != nil {
			return err
		}
		if base != "master" {
			err = gits.GitCmd(dir, "checkout", base)
			if err != nil {
				return err
			}
		}

		// TODO lets fork if required???
		/*
			pushGitURL, err := gits.GitCreatePushURL(gitURL, details.User)
			if err != nil {
				return err
			}
			err = gits.GitCmd(dir, "remote", "add", "upstream", forkEnvGitURL)
			if err != nil {
				return err
			}
			err = gits.GitCmd(dir, "remote", "add", "origin", pushGitURL)
			if err != nil {
				return err
			}
			err = gits.GitCmd(dir, "push", "-u", "origin", "master")
			if err != nil {
				return err
			}
		*/
	}
	branchNames, err := gits.GitGetRemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return fmt.Errorf("Failed to load remote branch names: %s", err)
	}
	o.Printf("Found remote branch names %s\n", strings.Join(branchNames, ", "))
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchName += "-" + string(uuid.NewUUID())
	}
	err = gits.GitCmd(dir, "branch", branchName)
	if err != nil {
		return err
	}
	err = gits.GitCmd(dir, "checkout", branchName)
	if err != nil {
		return err
	}

	requirementsFile, err := helm.FindRequirementsFileName(dir)
	if err != nil {
		return err
	}
	requirements, err := helm.LoadRequirementsFile(requirementsFile)
	if err != nil {
		return err
	}
	if version == "" {
		version, err = o.findLatestVersion(app)
		if err != nil {
			return err
		}
	}
	requirements.SetAppVersion(app, version, o.HelmRepositoryURL)
	err = helm.SaveRequirementsFile(requirementsFile, requirements)

	err = gits.GitCmd(dir, "add", "*", "*/*")
	if err != nil {
		return err
	}
	changed, err := gits.HasChanges(dir)
	if err != nil {
		return err
	}
	if !changed {
		o.Printf("%s\n", util.ColorWarning("No changes made to the GitOps Environment source code. Must be already on version!"))
		return nil
	}
	message := fmt.Sprintf("Promote %s to version %s", app, versionName)
	err = gits.GitCommit(dir, message)
	if err != nil {
		return err
	}
	err = gits.GitPush(dir)
	if err != nil {
		return err
	}

	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	provider, err := gitInfo.PickOrCreateProvider(authConfigSvc, "user name to submit the Pull Request", o.BatchMode)
	if err != nil {
		return err
	}

	gha := &gits.GitPullRequestArguments{
		Owner: gitInfo.Organisation,
		Repo:  gitInfo.Name,
		Title: app + " to " + versionName,
		Body:  message,
		Base:  base,
		Head:  branchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
		return err
	}
	o.Printf("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	releaseInfo.PullRequestInfo = &ReleasePullRequestInfo{
		GitProvider:          provider,
		PullRequest:          pr,
		PullRequestArguments: gha,
	}
	return nil
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

func (o *PromoteOptions) DiscoverAppName() (string, error) {
	answer := ""
	gitInfo, err := gits.GetGitInfo("")
	if err != nil {
		return answer, err
	}

	if gitInfo == nil {
		return answer, fmt.Errorf("no git info found to discover app name from")
	}
	answer = gitInfo.Name

	if answer == "" {
		chartFile, err := o.FindHelmChart()
		if err != nil {
			return answer, err
		}
		if chartFile != "" {
			return helm.LoadChartName(chartFile)
		}
	}
	return answer, nil
}

func (o *PromoteOptions) WaitForPromotion(ns string, env *v1.Environment, releaseInfo *ReleaseInfo) error {
	if o.TimeoutDuration == nil {
		o.Printf("No --%s option specified on the 'jx promote' command so not waiting for the promotion to succeed\n", optionTimeout)
		return nil
	}
	if o.PullRequestPollDuration == nil {
		o.Printf("No --%s option specified on the 'jx promote' command so not waiting for the promotion to succeed\n", optionPullRequestPollTime)
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
				return fmt.Errorf("Failed to query the Pull Request status for %s %s", pr.URL, err)
			}

			if pr.Merged != nil && *pr.Merged {
				if pr.MergeCommitSHA == nil {
					if !logNoMergeCommitSha {
						logNoMergeCommitSha = true
						o.Printf("Pull Request %s is merged but waiting for Merge SHA\n", util.ColorInfo(pr.URL))
					}
				} else {
					mergeSha := *pr.MergeCommitSHA
					if !logHasMergeSha {
						logHasMergeSha = true
						o.Printf("Pull Request %s is merged at sha %s\n", util.ColorInfo(pr.URL), util.ColorInfo(mergeSha))

						mergedPR := func(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
							kube.CompletePromotionPullRequest(a, s, ps, p)
							p.MergeCommitSHA = mergeSha
							return nil
						}
						promoteKey.OnPromotePullRequest(o.Activities, mergedPR)
					}

					promoteKey.OnPromoteUpdate(o.Activities, kube.StartPromotionUpdate)

					statuses, err := gitProvider.ListCommitStatus(pr.Owner, pr.Repo, mergeSha)
					if err != nil {
						if !logMergeStatusError {
							logMergeStatusError = true
							o.warnf("Failed to query merge status of repo %s/%s with merge sha %s due to: %s\n", pr.Owner, pr.Repo, mergeSha, err)
						}
					} else {
						if len(statuses) == 0 {
							if !logNoMergeStatuses {
								logNoMergeStatuses = true
								o.Printf("Merge commit has not yet any statuses on repo %s/%s merge sha %s\n", pr.Owner, pr.Repo, mergeSha)
							}
						} else {
							for _, status := range statuses {
								if status.IsFailed() {
									o.warnf("merge status: %s URL: %s description: %s\n",
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
										o.Printf("merge status: %s for URL %s with target: %s description: %s\n",
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
								o.Printf("Merge status checks all passed so the promotion worked!\n")
								err = promoteKey.OnPromoteUpdate(o.Activities, kube.CompletePromotionUpdate)
								if err == nil {
									err = o.commentOnIssues(ns, env)
								}
								return err
							}
						}
					}
				}
			} else {
				if pr.IsClosed() {
					o.warnf("Pull Request %s is closed\n", util.ColorInfo(pr.URL))
					return fmt.Errorf("Promotion failed as Pull Request %s is closed without merging", pr.URL)
				}

				// lets try merge if the status is good
				status, err := gitProvider.PullRequestLastCommitStatus(pr)
				if err != nil {
					o.warnf("Failed to query the Pull Request last commit status for %s ref %s %s\n", pr.URL, pr.LastCommitSha, err)
					//return fmt.Errorf("Failed to query the Pull Request last commit status for %s ref %s %s", pr.URL, pr.LastCommitSha, err)
				} else {
					if status == "success" {
						err = gitProvider.MergePullRequest(pr, "jx promote automatically merged promotion PR")
						if err != nil {
							if !logMergeFailure {
								logMergeFailure = true
								o.warnf("Failed to merge the Pull Request %s due to %s maybe I don't have karma?\n", pr.URL, err)
							}
						}
					} else if status == "error" || status == "failure" {
						return fmt.Errorf("Pull request %s last commit has status %s for ref %s", pr.URL, status, pr.LastCommitSha)
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
	output, err := o.getCommandOutput("", "helm", "search", app, "--versions")
	if err != nil {
		return "", err
	}
	var maxSemVer *semver.Version
	maxString := ""
	for i, line := range strings.Split(output, "\n") {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 1 {
			v := fields[1]
			if v != "" {
				sv, err := semver.Parse(v)
				if err != nil {
					o.warnf("Invalid semantic version: %s %s\n", v, err)
				} else {
					if maxSemVer == nil || maxSemVer.Compare(sv) > 0 {
						maxSemVer = &sv
					}
				}
				if maxString == "" || strings.Compare(v, maxString) > 0 {
					maxString = v
				}
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
		o.Printf("No helm home dir at %s so lets initialise helm client\n", helmHomeDir)

		err = o.runCommand("helm", "init", "--client-only")
		if err != nil {
			return err
		}
	}

	f := o.Factory
	_, ns, _ := f.CreateClient()
	if err != nil {
		return err
	}

	// lets add the releases chart
	return o.registerLocalHelmRepo(o.LocalHelmRepoName, ns)
}

func (o *PromoteOptions) createPromoteKey(env *v1.Environment) *kube.PromoteStepActivityKey {
	pipeline := os.Getenv("JOB_NAME")
	build := os.Getenv("BUILD_NUMBER")
	buildURL := os.Getenv("BUILD_URL")
	buildLogsURL := os.Getenv("BUILD_LOG_URL")
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
	o.Printf("Using pipeline name %s\n", name)
	gitInfo, err := gits.GetGitInfo("")
	if err != nil {
		o.warnf("Could not discover the git repository info %s\n", err)
	} else {
		o.GitInfo = gitInfo
	}
	return &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:         name,
			Pipeline:     pipeline,
			Build:        build,
			BuildURL:     buildURL,
			BuildLogsURL: buildLogsURL,
			GitInfo:      gitInfo,
		},
		Environment: env.Name,
	}
}

func (o *PromoteOptions) getJenkinsURL() string {
	if o.jenkinsURL == "" {
		o.jenkinsURL = os.Getenv("JENKINS_URL")
	}
	if o.jenkinsURL == "" {
		o.jenkinsURL = os.Getenv("JENKINS_URL")
	}
	url, err := o.Factory.GetJenkinsURL()
	if err != nil {
		o.warnf("Could not find Jenkins URL %s", err)
	} else {
		o.jenkinsURL = url
	}
	return o.jenkinsURL
}

// commentOnIssues comments on any issues for a release that the fix is available in the given environment
func (o *PromoteOptions) commentOnIssues(targetNS string, environment *v1.Environment) error {
	ens := environment.Spec.Namespace
	envName := environment.Spec.Label
	app := o.Application
	version := o.Version
	if ens == "" {
		o.warnf("Environment %s has no namespace\n", envName)
		return nil
	}
	if app == "" {
		o.warnf("No appplication name so cannot comment on issues that they are now in %s\n", envName)
		return nil
	}
	if version == "" {
		o.warnf("No version name so cannot comment on issues that they are now in %s\n", envName)
		return nil
	}
	gitInfo := o.GitInfo
	if gitInfo == nil {
		o.warnf("No GitInfo discovered so cannot comment on issues that they are now in %s\n", envName)
		return nil
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	provider, err := gitInfo.PickOrCreateProvider(authConfigSvc, "user name to comment on issues", o.BatchMode)
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
	release, err := jxClient.JenkinsV1().Releases(ens).Get(releaseName, metav1.GetOptions{})
	if err == nil && release != nil {
		issues := release.Spec.Issues

		available := ""
		ing, err := kubeClient.ExtensionsV1beta1().Ingresses(ens).Get(app, metav1.GetOptions{})
		if ing != nil && err == nil {
			if len(ing.Spec.Rules) > 0 {
				hostname := ing.Spec.Rules[0].Host
				if hostname != "" {
					available = fmt.Sprintf(" and available [here](http://%s)", hostname)
				}
			}
		}

		for _, issue := range issues {
			if issue.IsClosed() {
				o.Printf("Commenting that issue %s is now in %s\n", util.ColorInfo(issue.URL), util.ColorInfo(envName))

				comment := fmt.Sprintf(":white_check_mark: fix for issue %s is now deployed to %s%s", issue.URL, envName, available)
				id := issue.ID
				if id != "" {
					number, err := strconv.Atoi(id)
					if err != nil {
						o.warnf("Could not parse issue id %s for URL %s\n", id, issue.URL)
					} else {
						if number > 0 {
							err = provider.CreateIssueComment(gitInfo.Organisation, gitInfo.Name, number, comment)
							if err != nil {
								o.warnf("Failed to add comment to issue %s: %s", issue.URL, err)
							}
						}
					}
				}
			}
		}
	}
	return nil
}
