package cmd

import (
	"fmt"
	"os/user"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/environments"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/gits"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/util"
)

var (
	deleteApplicationLong = templates.LongDesc(`
		Deletes one or more Applications

		Note that this command does not remove the underlying Git Repositories. 

		For that see the [jx delete repo](https://jenkins-x.io/commands/jx_delete_repo/) command.

`)

	deleteApplicationExample = templates.Examples(`
		# prompt for the available applications to delete
		jx delete application 

		# delete a specific app 
		jx delete application cheese
	`)
)

// DeleteApplicationOptions are the flags for this delete commands
type DeleteApplicationOptions struct {
	*CommonOptions

	SelectAll           bool
	SelectFilter        string
	IgnoreEnvironments  bool
	NoMergePullRequest  bool
	Timeout             string
	PullRequestPollTime string
	Org                 string

	// calculated fields
	TimeoutDuration         *time.Duration
	PullRequestPollDuration *time.Duration

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback environments.ConfigureGitFn
}

// NewCmdDeleteApplication creates a command object for this command
func NewCmdDeleteApplication(commonOpts *CommonOptions) *cobra.Command {
	options := &DeleteApplicationOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "application",
		Short:   "Deletes one or more applications from Jenkins",
		Long:    deleteApplicationLong,
		Example: deleteApplicationExample,
		Aliases: []string{"applications"}, // FIXME - naming conflict with 'app'
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.IgnoreEnvironments, "no-env", "", false, "Do not remove the application from any of the Environments")
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "a", false, "Selects all the matched applications")
	cmd.Flags().BoolVarP(&options.NoMergePullRequest, "no-merge", "", false, "Disables automatic merge of promote Pull Requests")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "f", "", "Filter the list of applications to those containing this text")
	cmd.Flags().StringVarP(&options.Timeout, optionTimeout, "t", "1h", "The timeout to wait for the promotion to succeed in the underlying Environment. The command fails if the timeout is exceeded or the promotion does not complete")
	cmd.Flags().StringVarP(&options.PullRequestPollTime, optionPullRequestPollTime, "", "20s", "Poll time when waiting for a Pull Request to merge")
	cmd.Flags().StringVarP(&options.Org, "org", "o", "", "github organisation/project name that source code resides in")

	return cmd
}

// Run implements this command
func (o *DeleteApplicationOptions) Run() error {
	err := o.init()
	if err != nil {
		return errors.Wrap(err, "setting up context")
	}

	isProw, err := o.isProw()
	if err != nil {
		return errors.Wrap(err, "getting prow config")
	}

	repoService := kube.NewSourceRepositoryService(o.jxClient, o.currentNamespace)
	var deletedApplications []string
	if isProw {
		deletedApplications, err = o.deleteProwApplication(repoService)
	} else {
		deletedApplications, err = o.deleteJenkinsApplication()
	}

	if err != nil {
		return errors.Wrapf(err, "deleting application")
	}
	log.Infof("Deleted Application(s): %s\n", util.ColorInfo(strings.Join(deletedApplications, ",")))
	return nil
}

func (o *DeleteApplicationOptions) deleteProwApplication(repoService kube.SourceRepoer) (deletedApplications []string, err error) {
	envMap, _, err := kube.GetOrderedEnvironments(o.jxClient, "")
	currentUser, err := user.Current()
	if err != nil {
		return deletedApplications, errors.Wrap(err, "getting current user")
	}

	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return deletedApplications, errors.Wrap(err, "getting kube client")
	}

	for _, applicationName := range o.Args {
		for _, env := range envMap {
			if env.Spec.Kind == v1.EnvironmentKindTypePermanent {
				err = o.deleteApplicationFromEnvironment(env, applicationName, currentUser.Username)
				if err != nil {
					return deletedApplications, errors.Wrapf(err, "deleting application %s from environment %s", applicationName, env.Name)
				}
			}
		}
		if o.Org == "" {
			// fetch the list of sourcerepositories
			srList, err := repoService.ListSourceRepositories()
			if err != nil {
				return deletedApplications, errors.Wrapf(err, "error in sourcerepository service %s", err.Error())
			}

			srObjects := []v1.SourceRepository{}
			for sr := range srList.Items {
				if srList.Items[sr].Spec.Repo == applicationName {
					srObjects = append(srObjects, srList.Items[sr])
					if len(srObjects) > 1 {
						return deletedApplications, errors.Wrapf(err, "application %s exists in multiple orgs, use --org to specify the app to delete", util.ColorInfo(applicationName))
					}
				}
			}
			if len(srObjects) == 0 {
				return deletedApplications, errors.Wrapf(err, "unable to determine org for %s.  Please use --org to specify the app to delete", util.ColorInfo(applicationName))
			}

			// we only found a single sourceporistory resource, proceed
			o.Org = srObjects[0].Spec.Org
		}

		repo := []string{o.Org + "/" + applicationName}
		err = prow.DeleteApplication(kubeClient, repo, ns)
		if err != nil {
			return deletedApplications, errors.Wrapf(err, "deleting prow config for %s", applicationName)
		}
		deletedApplications = append(deletedApplications, applicationName)

		err := repoService.DeleteSourceRepository(o.Org + "-" + applicationName)
		if err != nil {
			log.Warnf("Unable to find application metadata for %s to remove", applicationName)
		}
	}
	return
}

func (o *DeleteApplicationOptions) deleteJenkinsApplication() (deletedApplications []string, err error) {
	args := o.Args

	jenk, err := o.JenkinsClient()
	if err != nil {
		return deletedApplications, err
	}

	jobs, err := jenkins.LoadAllJenkinsJobs(jenk)
	if err != nil {
		return deletedApplications, err
	}

	names := []string{}
	m := map[string]*gojenkins.Job{}

	for _, j := range jobs {
		if jenkins.IsMultiBranchProject(j) {
			name := j.FullName
			names = append(names, name)
			m[name] = j
		}
	}

	if len(names) == 0 {
		return deletedApplications, fmt.Errorf("There are no Applications in Jenkins")
	}

	if len(args) == 0 {
		args, err = util.SelectNamesWithFilter(names, "Pick Applications to remove from Jenkins:", o.SelectAll, o.SelectFilter, "", o.In, o.Out, o.Err)
		if err != nil {
			return deletedApplications, err
		}
		if len(args) == 0 {
			return deletedApplications, fmt.Errorf("No application was picked to be removed from Jenkins")
		}
	} else {
		for _, arg := range args {
			if util.StringArrayIndex(names, arg) < 0 {
				return deletedApplications, util.InvalidArg(arg, names)
			}
		}
	}
	deleteMessage := strings.Join(args, ", ")

	if !o.BatchMode {
		if !util.Confirm("You are about to delete these Applications from Jenkins: "+deleteMessage, false, "The list of Applications names to be deleted from Jenkins", o.In, o.Out, o.Err) {
			return deletedApplications, err
		}
	}
	for _, name := range args {
		job := m[name]
		if job != nil {
			err = o.deleteApplication(jenk, name, job)
			if err != nil {
				return deletedApplications, err
			}
			deletedApplications = append(deletedApplications, name)
		}
	}
	return deletedApplications, err
}

func (o *DeleteApplicationOptions) deleteApplication(jenkinsClient gojenkins.JenkinsClient, name string, job *gojenkins.Job) error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	envMap, envNames, err := kube.GetOrderedEnvironments(jxClient, ns)
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return err
	}

	applicationName := o.applicationNameFromJenkinsJobName(name)
	for _, envName := range envNames {
		// TODO filter on environment names?
		env := envMap[envName]
		if env != nil && env.Spec.Kind == v1.EnvironmentKindTypePermanent {
			err = o.deleteApplicationFromEnvironment(env, applicationName, u.Username)
			if err != nil {
				return err
			}
		}
	}

	// lets try delete the job from each environment first
	return jenkinsClient.DeleteJob(*job)
}

func (o *DeleteApplicationOptions) applicationNameFromJenkinsJobName(name string) string {
	path := strings.Split(name, "/")
	return path[len(path)-1]
}

func (o *DeleteApplicationOptions) deleteApplicationFromEnvironment(env *v1.Environment, applicationName string, username string) error {
	if o.IgnoreEnvironments {
		return nil
	}
	if env.Spec.Source.URL == "" {
		return nil
	}
	log.Infof("Removing application %s from environment %s\n", applicationName, env.Spec.Label)

	modifyChartFn := func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
		templates map[string]string, dir string, info *environments.PullRequestDetails) error {
		requirements.RemoveApplication(applicationName)
		return nil
	}
	gitProvider, _, err := o.createGitProviderForURLWithoutKind(env.Spec.Source.URL)
	if err != nil {
		return errors.Wrapf(err, "creating git provider for %s", env.Spec.Source.URL)
	}
	environmentsDir, err := o.EnvironmentsDir()
	if err != nil {
		return errors.Wrapf(err, "getting environments dir")
	}

	details := environments.PullRequestDetails{
		BranchName: "delete-" + applicationName,
		Title:      "Delete application " + applicationName + " from this environment",
		Message:    "The command `jx delete application` was run by " + username + " and it generated this Pull Request",
	}
	options := environments.EnvironmentPullRequestOptions{
		ConfigGitFn:   o.ConfigureGitCallback,
		Gitter:        o.Git(),
		ModifyChartFn: modifyChartFn,
		GitProvider:   gitProvider,
	}
	info, err := options.Create(env, environmentsDir, &details, nil)
	if err != nil {
		return err
	}

	duration := *o.TimeoutDuration
	end := time.Now().Add(duration)

	return o.waitForGitOpsPullRequest(env, info, options.GitProvider, end, duration)
}

func (o *DeleteApplicationOptions) waitForGitOpsPullRequest(env *v1.Environment,
	pullRequestInfo *gits.PullRequestInfo, gitProvider gits.GitProvider, end time.Time,
	duration time.Duration) error {
	if pullRequestInfo != nil {
		logMergeFailure := false
		pr := pullRequestInfo.PullRequest
		log.Infof("Waiting for pull request %s to merge\n", pr.URL)

		for {
			err := gitProvider.UpdatePullRequestStatus(pr)
			if err != nil {
				return fmt.Errorf("Failed to query the Pull Request status for %s %s", pr.URL, err)
			}

			if pr.Merged != nil && *pr.Merged {
				log.Infof("Request %s is merged!\n", util.ColorInfo(pr.URL))
				return nil
			} else {
				if pr.IsClosed() {
					log.Warnf("Pull Request %s is closed\n", util.ColorInfo(pr.URL))
					return fmt.Errorf("Promotion failed as Pull Request %s is closed without merging", pr.URL)
				}
				// lets try merge if the status is good
				status, err := gitProvider.PullRequestLastCommitStatus(pr)
				if err != nil {
					log.Warnf("Failed to query the Pull Request last commit status for %s ref %s %s\n", pr.URL, pr.LastCommitSha, err)
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
			if time.Now().After(end) {
				return fmt.Errorf("Timed out waiting for pull request %s to merge. Waited %s", pr.URL, duration.String())
			}
			time.Sleep(*o.PullRequestPollDuration)
		}
	}
	return nil
}

func (o *DeleteApplicationOptions) init() error {
	var err error
	o.jxClient, o.currentNamespace, err = o.JXClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "getting jx client")
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
	return nil
}
