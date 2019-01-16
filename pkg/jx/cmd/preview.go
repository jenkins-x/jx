package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/users"

	"github.com/jenkins-x/jx/pkg/kube/services"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	previewLong = templates.LongDesc(`
		Creates or updates a Preview Environment for the given Pull Request or Branch.

		For more documentation on Preview Environments see: [https://jenkins-x.io/about/features/#preview-environments](https://jenkins-x.io/about/features/#preview-environments)

`)

	previewExample = templates.Examples(`
		# Create or updates the Preview Environment for the Pull Request
		jx preview
	`)
)

const (
	DOCKER_REGISTRY                        = "DOCKER_REGISTRY"
	JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST = "JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST"
	JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT = "JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT"
	ORG                                    = "ORG"
	APP_NAME                               = "APP_NAME"
	DOCKER_REGISTRY_ORG                    = "DOCKER_REGISTRY_ORG"
	PREVIEW_VERSION                        = "PREVIEW_VERSION"

	optionPostPreviewJobTimeout  = "post-preview-job-timeout"
	optionPostPreviewJobPollTime = "post-preview-poll-time"
)

// PreviewOptions the options for viewing running PRs
type PreviewOptions struct {
	PromoteOptions

	Name                   string
	Label                  string
	Namespace              string
	DevNamespace           string
	Cluster                string
	PullRequestURL         string
	PullRequest            string
	SourceURL              string
	SourceRef              string
	Dir                    string
	PostPreviewJobTimeout  string
	PostPreviewJobPollTime string

	PullRequestName string
	GitConfDir      string
	GitProvider     gits.GitProvider
	GitInfo         *gits.GitRepository
	NoComment       bool

	// calculated fields
	PostPreviewJobTimeoutDuration time.Duration
	PostPreviewJobPollDuration    time.Duration

	HelmValuesConfig config.HelmValuesConfig
}

// NewCmdPreview creates a command object for the "create" command
func NewCmdPreview(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &PreviewOptions{
		HelmValuesConfig: config.HelmValuesConfig{
			ExposeController: &config.ExposeController{},
		},
		PromoteOptions: PromoteOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "preview",
		Short:   "Creates or updates a Preview Environment for the current version of an application",
		Long:    previewLong,
		Example: previewExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			//Default to batch-mode when running inside the pipeline (but user override wins).
			if !cmd.Flag(optionBatchMode).Changed {
				options.BatchMode = options.IsInCDPipeline()
			}
			err := options.Run()
			CheckErr(err)
		},
	}
	//addCreateAppFlags(cmd, &options.CreateOptions)

	options.addPreviewOptions(cmd)
	options.addCommonFlags(cmd)
	options.HelmValuesConfig.AddExposeControllerValues(cmd, false)
	options.PromoteOptions.addPromoteOptions(cmd)

	return cmd
}

func (options *PreviewOptions) addPreviewOptions(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&options.Name, kube.OptionName, "n", "", "The Environment resource name. Must follow the Kubernetes name conventions like Services, Namespaces")
	cmd.Flags().StringVarP(&options.Label, "label", "l", "", "The Environment label which is a descriptive string like 'Production' or 'Staging'")
	cmd.Flags().StringVarP(&options.Namespace, kube.OptionNamespace, "", "", "The Kubernetes namespace for the Environment")
	cmd.Flags().StringVarP(&options.DevNamespace, "dev-namespace", "", "", "The Developer namespace where the preview command should run")
	cmd.Flags().StringVarP(&options.Cluster, "cluster", "c", "", "The Kubernetes cluster for the Environment. If blank and a namespace is specified assumes the current cluster")
	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The source directory used to detect the git source URL and reference")
	cmd.Flags().StringVarP(&options.PullRequest, "pr", "", "", "The Pull Request Name (e.g. 'PR-23' or just '23'")
	cmd.Flags().StringVarP(&options.PullRequestURL, "pr-url", "", "", "The Pull Request URL")
	cmd.Flags().StringVarP(&options.SourceURL, "source-url", "s", "", "The source code git URL")
	cmd.Flags().StringVarP(&options.SourceRef, "source-ref", "", "", "The source code git ref (branch/sha)")
	cmd.Flags().StringVarP(&options.PostPreviewJobTimeout, optionPostPreviewJobTimeout, "", "2h", "The duration before we consider the post preview Jobs failed")
	cmd.Flags().StringVarP(&options.PostPreviewJobPollTime, optionPostPreviewJobPollTime, "", "10s", "The amount of time between polls for the post preview Job status")
	cmd.Flags().BoolVarP(&options.NoComment, "no-comment", "", false, "Disables commenting on the Pull Request after preview is created.")
}

// Run implements the command
func (o *PreviewOptions) Run() error {
	var err error
	if o.PostPreviewJobPollTime != "" {
		o.PostPreviewJobPollDuration, err = time.ParseDuration(o.PostPreviewJobPollTime)
		if err != nil {
			return fmt.Errorf("Invalid duration format %s for option --%s: %s", o.PostPreviewJobPollTime, optionPostPreviewJobPollTime, err)
		}
	}
	if o.PostPreviewJobTimeout != "" {
		o.PostPreviewJobTimeoutDuration, err = time.ParseDuration(o.Timeout)
		if err != nil {
			return fmt.Errorf("Invalid duration format %s for option --%s: %s", o.Timeout, optionPostPreviewJobTimeout, err)
		}
	}

	log.Infoln("Creating a preview")
	/*
		args := o.Args
		if len(args) > 0 && o.Name == "" {
			o.Name = args[0]
		}
	*/
	jxClient, currentNs, err := o.JXClient()
	if err != nil {
		return err
	}
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}
	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterEnvironmentCRD(apisClient)
	if err != nil {
		return err
	}
	err = kube.RegisterGitServiceCRD(apisClient)
	if err != nil {
		return err
	}
	err = kube.RegisterUserCRD(apisClient)
	if err != nil {
		return err
	}

	ns := o.DevNamespace
	if ns == "" {
		ns, _, err = kube.GetDevNamespace(kubeClient, currentNs)
		if err != nil {
			return err
		}
	}
	o.DevNamespace = ns

	err = o.defaultValues(ns, true)
	if err != nil {
		return err
	}

	if o.GitInfo == nil {
		log.Warnf("No GitInfo found\n")
	} else if o.GitInfo.Organisation == "" {
		log.Warnf("No GitInfo.Organisation found\n")
	} else if o.GitInfo.Name == "" {
		log.Warnf("No GitInfo.Name found\n")
	}

	// we need pull request info to include
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}

	prNum, err := strconv.Atoi(o.PullRequestName)
	if err != nil {
		log.Warn("Unable to convert PR " + o.PullRequestName + " to a number" + "\n")
	}

	var user *v1.UserSpec
	buildStatus := ""
	buildStatusUrl := ""

	var pullRequest *gits.GitPullRequest

	if o.GitInfo != nil {
		gitKind, err := o.GitServerKind(o.GitInfo)
		if err != nil {
			return err
		}

		gitProvider, err := o.CreateGitProvider(o.GitInfo.URL, "message", authConfigSvc, gitKind, o.BatchMode, o.Git(), o.In, o.Out, o.Err)
		if err != nil {
			return fmt.Errorf("cannot create Git provider %v", err)
		}

		resolver := users.GitUserResolver{
			GitProvider: gitProvider,
			JXClient:    jxClient,
			Namespace:   currentNs,
		}

		if prNum > 0 {
			pullRequest, err = gitProvider.GetPullRequest(o.GitInfo.Organisation, o.GitInfo, prNum)
			if err != nil {
				log.Warnf("issue getting pull request %s, %s, %v: %v\n", o.GitInfo.Organisation, o.GitInfo.Name, prNum, err)
			}
			commits, err := gitProvider.GetPullRequestCommits(o.GitInfo.Organisation, o.GitInfo, prNum)
			if err != nil {
				log.Warn("Unable to get commits: " + err.Error() + "\n")
			}
			if pullRequest != nil {
				author, err := resolver.Resolve(pullRequest.Author)
				if err != nil {
					return err
				}
				author, err = resolver.UpdateUserFromPRAuthor(author, pullRequest, commits)
				if err != nil {
					// This isn't fatal, just nice to have!
					log.Warnf("Unable to update user %s from %s because %v", author.Name, o.PullRequestName, err)
				}
				if author != nil {
					user = &v1.UserSpec{
						Username: author.Spec.Login,
						Name:     author.Spec.Name,
						ImageURL: author.Spec.AvatarURL,
						LinkURL:  author.Spec.URL,
					}
				}
			}

			statuses, err := gitProvider.ListCommitStatus(o.GitInfo.Organisation, o.GitInfo.Name, pullRequest.LastCommitSha)

			if err != nil {
				log.Warn("Unable to get statuses for PR " + o.PullRequestName + "\n")
			}

			if len(statuses) > 0 {
				status := statuses[len(statuses)-1]
				buildStatus = status.State
				buildStatusUrl = status.TargetURL
			}
		}
	}

	environmentsResource := jxClient.JenkinsV1().Environments(ns)
	env, err := environmentsResource.Get(o.Name, metav1.GetOptions{})
	if err == nil {
		// lets check for updates...
		update := false

		spec := &env.Spec
		source := &spec.Source
		if spec.Label != o.Label {
			spec.Label = o.Label
			update = true
		}
		if spec.Namespace != o.Namespace {
			spec.Namespace = o.Namespace
			update = true
		}
		if spec.Namespace != o.Namespace {
			spec.Namespace = o.Namespace
			update = true
		}
		if spec.Kind != v1.EnvironmentKindTypePreview {
			spec.Kind = v1.EnvironmentKindTypePreview
			update = true
		}
		if source.Kind != v1.EnvironmentRepositoryTypeGit {
			source.Kind = v1.EnvironmentRepositoryTypeGit
			update = true
		}
		if source.URL != o.SourceURL {
			source.URL = o.SourceURL
			update = true
		}
		if source.Ref != o.SourceRef {
			source.Ref = o.SourceRef
			update = true
		}

		gitSpec := spec.PreviewGitSpec
		if gitSpec.BuildStatus != buildStatus {
			gitSpec.BuildStatus = buildStatus
			update = true
		}
		if gitSpec.BuildStatusURL != buildStatusUrl {
			gitSpec.BuildStatusURL = buildStatusUrl
			update = true
		}
		if gitSpec.ApplicationName != o.Application {
			gitSpec.ApplicationName = o.Application
			update = true
		}
		if pullRequest != nil {
			if gitSpec.Title != pullRequest.Title {
				gitSpec.Title = pullRequest.Title
				update = true
			}
			if gitSpec.Description != pullRequest.Body {
				gitSpec.Description = pullRequest.Body
				update = true
			}
		}
		if gitSpec.URL != o.PullRequestURL {
			gitSpec.URL = o.PullRequestURL
			update = true
		}
		if user != nil {
			if gitSpec.User.Username != user.Username ||
				gitSpec.User.ImageURL != user.ImageURL ||
				gitSpec.User.Name != user.Name ||
				gitSpec.User.LinkURL != user.LinkURL {
				gitSpec.User = *user
				update = true
			}
		}

		if update {
			env, err = environmentsResource.Update(env)
			if err != nil {
				return fmt.Errorf("Failed to update Environment %s due to %s", o.Name, err)
			}
		}
	} else {
		// lets create a new preview environment
		previewGitSpec := v1.PreviewGitSpec{
			ApplicationName: o.Application,
			Name:            o.PullRequestName,
			URL:             o.PullRequestURL,
			BuildStatus:     buildStatus,
			BuildStatusURL:  buildStatusUrl,
		}
		if pullRequest != nil {
			previewGitSpec.Title = pullRequest.Title
			previewGitSpec.Description = pullRequest.Body
		}
		if user != nil {
			previewGitSpec.User = *user
		}
		env = &v1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: o.Name,
			},
			Spec: v1.EnvironmentSpec{
				Namespace:         o.Namespace,
				Label:             o.Label,
				Kind:              v1.EnvironmentKindTypePreview,
				PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
				PullRequestURL:    o.PullRequestURL,
				Order:             999,
				Source: v1.EnvironmentRepository{
					Kind: v1.EnvironmentRepositoryTypeGit,
					URL:  o.SourceURL,
					Ref:  o.SourceRef,
				},
				PreviewGitSpec: previewGitSpec,
			},
		}
		_, err = environmentsResource.Create(env)
		if err != nil {
			return fmt.Errorf("Failed to create environment in namespace %s due to: %s", ns, err)
		}
		log.Infof("Created environment %s\n", util.ColorInfo(env.Name))
	}

	err = kube.EnsureEnvironmentNamespaceSetup(kubeClient, jxClient, env, ns)
	if err != nil {
		return err
	}

	if o.ReleaseName == "" {
		_, noTiller, helmTemplate, err := o.TeamHelmBin()
		if err != nil {
			return err
		}
		if noTiller || helmTemplate {
			o.ReleaseName = "preview"
		} else {
			o.ReleaseName = o.Namespace
		}
	}

	domain, err := kube.GetCurrentDomain(kubeClient, ns)
	if err != nil {
		return err
	}

	values, err := o.GetPreviewValuesConfig(domain)
	if err != nil {
		return err
	}

	config, err := values.String()
	if err != nil {
		return err
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	configFileName := filepath.Join(dir, ExtraValuesFile)
	log.Infof("%s", config)
	err = ioutil.WriteFile(configFileName, []byte(config), 0644)
	if err != nil {
		return err
	}

	err = o.Helm().UpgradeChart(".", o.ReleaseName, o.Namespace, nil, true, nil, true, true, nil,
		[]string{configFileName}, "", "", "")
	if err != nil {
		return err
	}

	url := ""
	appNames := []string{o.Application, o.ReleaseName, o.Namespace + "-preview", o.ReleaseName + "-" + o.Application}
	for _, n := range appNames {
		url, err = services.FindServiceURL(kubeClient, o.Namespace, n)
		if url != "" {
			writePreviewURL(o, url)
			break
		}
	}

	if url == "" {
		log.Warnf("Could not find the service URL in namespace %s for names %s\n", o.Namespace, strings.Join(appNames, ", "))
	}

	comment := fmt.Sprintf(":star: PR built and available in a preview environment **%s**", o.Name)
	if url != "" {
		comment += fmt.Sprintf(" [here](%s) ", url)
	}

	pipeline := o.getJobName()
	build := o.getBuildNumber()

	if url != "" || o.PullRequestURL != "" {
		if pipeline != "" && build != "" {
			name := kube.ToValidName(pipeline + "-" + build)
			// lets see if we can update the pipeline
			activities := jxClient.JenkinsV1().PipelineActivities(ns)
			key := &kube.PromoteStepActivityKey{
				PipelineActivityKey: kube.PipelineActivityKey{
					Name:     name,
					Pipeline: pipeline,
					Build:    build,
				},
			}
			a, _, p, _, err := key.GetOrCreatePreview(o.jxClient, o.Namespace)
			if err == nil && a != nil && p != nil {
				updated := false
				if p.ApplicationURL == "" {
					p.ApplicationURL = url
					updated = true
				}
				if p.PullRequestURL == "" && o.PullRequestURL != "" {
					p.PullRequestURL = o.PullRequestURL
					updated = true
				}
				if updated {
					_, err = activities.Update(a)
					if err != nil {
						log.Warnf("Failed to update PipelineActivities %s: %s\n", name, err)
					} else {
						log.Infof("Updating PipelineActivities %s which has status %s\n", name, string(a.Spec.Status))
					}
				}
			}
		} else {
			log.Warnf("No pipeline and build number available on $JOB_NAME and $BUILD_NUMBER so cannot update PipelineActivities with the preview URLs\n")
		}
	}
	if url != "" {
		env, err = environmentsResource.Get(o.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if env != nil && env.Spec.PreviewGitSpec.ApplicationURL == "" {
			env.Spec.PreviewGitSpec.ApplicationURL = url
			_, err = environmentsResource.Update(env)
			if err != nil {
				return fmt.Errorf("Failed to update Environment %s due to %s", o.Name, err)
			}
		}
		log.Infof("Preview application is now available at: %s\n\n", util.ColorInfo(url))
	}

	stepPRCommentOptions := StepPRCommentOptions{
		Flags: StepPRCommentFlags{
			Owner:      o.GitInfo.Organisation,
			Repository: o.GitInfo.Name,
			Comment:    comment,
			PR:         o.PullRequestName,
		},
		StepPROptions: StepPROptions{
			StepOptions: StepOptions{
				CommonOptions: o.CommonOptions,
			},
		},
	}
	stepPRCommentOptions.BatchMode = true
	if !o.NoComment {
		err = stepPRCommentOptions.Run()
	}
	if err != nil {
		log.Warnf("Failed to comment on the Pull Request with owner %s repo %s: %s\n", o.GitInfo.Organisation, o.GitInfo.Name, err)
	}
	return o.RunPostPreviewSteps(kubeClient, o.Namespace, url, pipeline, build)
}

// RunPostPreviewSteps lets run any post-preview steps that are configured for all apps in a team
func (o *PreviewOptions) RunPostPreviewSteps(kubeClient kubernetes.Interface, ns string, url string, pipeline string, build string) error {
	teamSettings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	envVars := map[string]string{
		"JX_PREVIEW_URL": url,
		"JX_PIPELINE":    pipeline,
		"JX_BUILD":       build,
	}

	jobs := teamSettings.PostPreviewJobs
	jobResources := kubeClient.BatchV1().Jobs(ns)
	createdJobs := []*batchv1.Job{}
	for _, job := range jobs {
		// TODO lets modify the job name?
		job2 := o.modifyJob(&job, envVars)
		log.Infof("Triggering post preview Job %s in namespace %s\n", util.ColorInfo(job2.Name), util.ColorInfo(ns))

		gracePeriod := int64(0)
		propationPolicy := metav1.DeletePropagationForeground

		// lets try delete it if it exists
		jobResources.Delete(job2.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			PropagationPolicy:  &propationPolicy,
		})

		// lets wait for the resource to be gone
		hasJob := func() (bool, error) {
			job, err := jobResources.Get(job.Name, metav1.GetOptions{})
			return job == nil || err != nil, nil
		}
		o.retryUntilTrueOrTimeout(time.Minute, time.Second, hasJob)

		createdJob, err := jobResources.Create(job2)
		if err != nil {
			return err
		}
		createdJobs = append(createdJobs, createdJob)
	}
	return o.waitForJobsToComplete(kubeClient, createdJobs)
}

func (o *PreviewOptions) waitForJobsToComplete(kubeClient kubernetes.Interface, jobs []*batchv1.Job) error {
	for _, job := range jobs {
		err := o.waitForJob(kubeClient, job)
		if err != nil {
			return err
		}
	}
	return nil
}

// waits for this job to complete
func (o *PreviewOptions) waitForJob(kubeClient kubernetes.Interface, job *batchv1.Job) error {
	name := job.Name
	ns := job.Namespace
	log.Infof("waiting for Job %s in namespace %s to complete...\n\n", util.ColorInfo(name), util.ColorInfo(ns))

	count := 0
	fn := func() (bool, error) {
		curJob, err := kubeClient.BatchV1().Jobs(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		if kube.IsJobFinished(curJob) {
			if kube.IsJobSucceeded(curJob) {
				return true, nil
			} else {
				failed := curJob.Status.Failed
				succeeded := curJob.Status.Succeeded
				return true, fmt.Errorf("Job %s in namepace %s has %d failed containers and %d succeeded containers", name, ns, failed, succeeded)
			}
		}
		count += 1
		if count > 1 {
			// TODO we could maybe do better - using a prefix on all logs maybe with the job name?
			o.runCommandVerbose("kubectl", "logs", "-f", "job/"+name, "-n", ns)
		}
		return false, nil
	}
	err := o.retryUntilTrueOrTimeout(o.PostPreviewJobTimeoutDuration, o.PostPreviewJobPollDuration, fn)
	if err != nil {
		log.Warnf("\nFailed to complete post Preview Job %s in namespace %s: %s\n", name, ns, err)
	}
	return err
}

// modifyJob adds the given environment variables into all the containers in the job
func (o *PreviewOptions) modifyJob(originalJob *batchv1.Job, envVars map[string]string) *batchv1.Job {
	job := *originalJob
	for k, v := range envVars {
		templateSpec := &job.Spec.Template.Spec
		for i, _ := range templateSpec.Containers {
			container := &templateSpec.Containers[i]
			if kube.GetEnvVar(container, k) == nil {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  k,
					Value: v,
				})
			}
		}
	}

	return &job
}

func (o *PreviewOptions) defaultValues(ns string, warnMissingName bool) error {
	var err error
	if o.Application == "" {
		o.Application, err = o.DiscoverAppName()
		if err != nil {
			return err
		}
	}

	// fill in default values
	if o.SourceURL == "" {
		o.SourceURL = os.Getenv("SOURCE_URL")
		if o.SourceURL == "" {
			// Relevant in a Jenkins pipeline triggered by a PR
			o.SourceURL = os.Getenv("CHANGE_URL")
			if o.SourceURL == "" {
				// lets discover the git dir
				if o.Dir == "" {
					dir, err := os.Getwd()
					if err != nil {
						return err
					}
					o.Dir = dir
				}
				root, gitConf, err := o.Git().FindGitConfigDir(o.Dir)
				if err != nil {
					log.Warnf("Could not find a .git directory: %s\n", err)
				} else {
					if root != "" {
						o.Dir = root
						o.SourceURL, err = o.discoverGitURL(gitConf)
						if err != nil {
							log.Warnf("Could not find the remote git source URL:  %s\n", err)
						} else {
							if o.SourceRef == "" {
								o.SourceRef, err = o.Git().Branch(root)
								if err != nil {
									log.Warnf("Could not find the remote git source ref:  %s\n", err)
								}

							}
						}
					}
				}
			}
		}
	}

	if o.SourceURL == "" {
		return fmt.Errorf("No sourceURL could be defaulted for the Preview Environment. Use --dir flag to detect the git source URL")
	}

	if o.PullRequest == "" {
		o.PullRequest = os.Getenv("BRANCH_NAME")
	}

	o.PullRequestName = strings.TrimPrefix(o.PullRequest, "PR-")

	if o.SourceURL != "" {
		o.GitInfo, err = gits.ParseGitURL(o.SourceURL)
		if err != nil {
			log.Warnf("Could not parse the git URL %s due to %s\n", o.SourceURL, err)
		} else {
			o.SourceURL = o.GitInfo.HttpCloneURL()
			if o.PullRequestURL == "" {
				if o.PullRequest == "" {
					if warnMissingName {
						log.Warnf("No Pull Request name or URL specified nor could one be found via $BRANCH_NAME\n")
					}
				} else {
					o.PullRequestURL = o.GitInfo.PullRequestURL(o.PullRequestName)
				}
			}
			if o.Name == "" && o.PullRequestName != "" {
				o.Name = o.GitInfo.Organisation + "-" + o.GitInfo.Name + "-pr-" + o.PullRequestName
			}
			if o.Label == "" {
				o.Label = o.GitInfo.Organisation + "/" + o.GitInfo.Name + " PR-" + o.PullRequestName
			}
		}
	}
	o.Name = kube.ToValidName(o.Name)
	if o.Name == "" {
		return fmt.Errorf("No name could be defaulted for the Preview Environment. Please supply one!")
	}
	if o.Namespace == "" {
		prefix := ns + "-"
		if len(prefix) > 63 {
			return fmt.Errorf("Team namespace prefix is too long to create previews %s is too long. Must be no more than 60 character", prefix)
		}

		o.Namespace = prefix + o.Name
		if len(o.Namespace) > 63 {
			max := 62 - len(prefix)
			size := len(o.Name)

			o.Namespace = prefix + o.Name[size-max:]
			log.Warnf("Due the name of the organsation and repository being too long (%s) we are going to trim it to make the preview namespace: %s", o.Name, o.Namespace)
		}
	}
	if len(o.Namespace) > 63 {
		return fmt.Errorf("Preview namespace %s is too long. Must be no more than 63 character", o.Namespace)
	}
	o.Namespace = kube.ToValidName(o.Namespace)
	if o.Label == "" {
		o.Label = o.Name
	}
	if o.GitInfo == nil {
		log.Warnf("No GitInfo could be found!")
	}
	return nil
}

// GetPreviewValuesConfig returns the PreviewValuesConfig to use as extraValues for helm
func (o *PreviewOptions) GetPreviewValuesConfig(domain string) (*config.PreviewValuesConfig, error) {
	repository, err := getImageName()
	if err != nil {
		return nil, err
	}

	tag, err := getImageTag()
	if err != nil {
		return nil, err
	}

	if o.HelmValuesConfig.ExposeController == nil {
		o.HelmValuesConfig.ExposeController = &config.ExposeController{}
	}
	o.HelmValuesConfig.ExposeController.Config.Domain = domain

	values := config.PreviewValuesConfig{
		ExposeController: o.HelmValuesConfig.ExposeController,
		Preview: &config.Preview{
			Image: &config.Image{
				Repository: repository,
				Tag:        tag,
			},
		},
	}
	return &values, nil
}

func writePreviewURL(o *PreviewOptions, url string) {
	previewFileName := filepath.Join(o.Dir, ".previewUrl")
	err := ioutil.WriteFile(previewFileName, []byte(url), 0644)
	if err != nil {
		log.Warn("Unable to write preview file")
	}
}

func getContainerRegistry() (string, error) {
	registry := os.Getenv(DOCKER_REGISTRY)
	if registry != "" {
		return registry, nil
	}

	registryHost := os.Getenv(JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST)
	if registryHost == "" {
		return "", fmt.Errorf("no %s environment variable found", JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST)
	}
	registryPort := os.Getenv(JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT)
	if registryPort == "" {
		return "", fmt.Errorf("no %s environment variable found", JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT)
	}

	return fmt.Sprintf("%s:%s", registryHost, registryPort), nil
}

func getImageName() (string, error) {
	containerRegistry, err := getContainerRegistry()
	if err != nil {
		return "", err
	}

	organisation := os.Getenv(ORG)
	if organisation == "" {
		return "", fmt.Errorf("no %s environment variable found", ORG)
	}

	app := os.Getenv(APP_NAME)
	if app == "" {
		return "", fmt.Errorf("no %s environment variable found", APP_NAME)
	}

	dockerRegistryOrg := os.Getenv(DOCKER_REGISTRY_ORG)
	if dockerRegistryOrg == "" {
		dockerRegistryOrg = organisation
	}

	return fmt.Sprintf("%s/%s/%s", containerRegistry, dockerRegistryOrg, app), nil
}

func getImageTag() (string, error) {
	tag := os.Getenv(PREVIEW_VERSION)
	if tag == "" {
		return "", fmt.Errorf("no %s environment variable found", PREVIEW_VERSION)
	}
	return tag, nil
}
