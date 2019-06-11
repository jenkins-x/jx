package gc

import (
	"strings"
	"time"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	jv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tv1alpha1 "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GCActivitiesOptions struct {
	*opts.CommonOptions

	DryRun                  bool
	ReleaseHistoryLimit     int
	PullRequestHistoryLimit int
	ReleaseAgeLimit         time.Duration
	PullRequestAgeLimit     time.Duration
	jclient                 gojenkins.JenkinsClient
}

var (
	GCActivitiesLong = templates.LongDesc(`
		Garbage collect the Jenkins X PipelineActivity and PipelineRun resources

`)

	GCActivitiesExample = templates.Examples(`
		# garbage collect PipelineActivity and PipelineRun resources
		jx gc activities

		# dry run mode
		jx gc pa --dry-run
`)
)

type buildCounter struct {
	ReleaseCount int
	PRCount      int
}

type buildsCount struct {
	cache map[string]*buildCounter
}

// AddBuild adds the build and returns the number of builds for this repo and branch
func (c *buildsCount) AddBuild(repoAndBranch string, isPR bool) int {
	if c.cache == nil {
		c.cache = map[string]*buildCounter{}
	}
	bc := c.cache[repoAndBranch]
	if bc == nil {
		bc = &buildCounter{}
		c.cache[repoAndBranch] = bc
	}
	if isPR {
		bc.PRCount++
		return bc.PRCount
	}
	bc.ReleaseCount++
	return bc.ReleaseCount
}

// NewCmd s a command object for the "step" command
func NewCmdGCActivities(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GCActivitiesOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "activities",
		Aliases: []string{"pa", "act", "pr"},
		Short:   "garbage collection for PipelineActivities and PipelineRun resources",
		Long:    GCActivitiesLong,
		Example: GCActivitiesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.DryRun, "dry-run", "d", false, "Dry run mode. If enabled just list the resources that would be removed")
	cmd.Flags().IntVarP(&options.ReleaseHistoryLimit, "release-history-limit", "l", 5, "Maximum number of PipelineActivities and PipelineRuns to keep around per repository release")
	cmd.Flags().IntVarP(&options.PullRequestHistoryLimit, "pr-history-limit", "", 2, "Minimum number of PipelineActivities and PipelineRuns to keep around per repository Pull Request")
	cmd.Flags().DurationVarP(&options.ReleaseAgeLimit, "pull-request-age", "p", time.Hour*48, "Maximum age to keep PipelineActivities and PipelineRun's for Pull Requests")
	cmd.Flags().DurationVarP(&options.PullRequestAgeLimit, "release-age", "r", time.Hour*24*30, "Maximum age to keep PipelineActivities and PipelineRun's for Releases")
	return cmd
}

// Run implements this command
func (o *GCActivitiesOptions) Run() error {
	client, currentNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	prowEnabled, err := o.IsProw()
	if err != nil {
		return err
	}

	if prowEnabled {
		err = o.gcPipelineRuns(client, currentNs)
		if err != nil {
			return err
		}
	}

	// cannot use field selectors like `spec.kind=Preview` on CRDs so list all environments
	activityInterface := client.JenkinsV1().PipelineActivities(currentNs)
	activities, err := activityInterface.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(activities.Items) == 0 {
		// no preview environments found so lets return gracefully
		log.Logger().Debug("no activities found")
		return nil
	}

	var jobNames []string
	if !prowEnabled {
		o.jclient, err = o.JenkinsClient()
		if err != nil {
			return err
		}

		jobs, err := o.jclient.GetJobs()
		if err != nil {
			return err
		}
		for _, j := range jobs {
			err = o.GetAllPipelineJobNames(o.jclient, &jobNames, j.Name)
			if err != nil {
				return err
			}
		}
	}

	now := time.Now()
	counters := &buildsCount{}

	// use reverse order so we remove the oldest ones
	for i := len(activities.Items) - 1; i >= 0; i-- {
		a := activities.Items[i]
		branchName := a.BranchName()
		isPR, isBatch := o.isPullRequestOrBatchBranch(branchName)
		maxAge, revisionHistory := o.ageAndHistoryLimits(isPR, isBatch)
		// lets remove activities that are too old
		if a.Spec.CompletedTimestamp != nil && a.Spec.CompletedTimestamp.Add(maxAge).Before(now) {
			err = o.deleteActivity(activityInterface, &a)
			if err != nil {
				return err
			}
			continue
		}

		repoAndBranchName := a.RepositoryOwner() + "/" + a.RepositoryName() + "/" + a.BranchName()
		c := counters.AddBuild(repoAndBranchName, isPR)
		if c > revisionHistory {
			err = o.deleteActivity(activityInterface, &a)
			if err != nil {
				return err
			}
			continue
		}

		if !prowEnabled {
			// if activity has no job in Jenkins delete it
			matched := false
			for _, j := range jobNames {
				if a.Spec.Pipeline == j {
					matched = true
					break
				}
			}
			if !matched {
				err = o.deleteActivity(activityInterface, &a)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (o *GCActivitiesOptions) gcPipelineRuns(jxClient versioned.Interface, ns string) error {
	tektonkClient, _, err := o.TektonClient()
	if err != nil {
		return err
	}
	pipelineRunInterface := tektonkClient.TektonV1alpha1().PipelineRuns(ns)
	activities, err := pipelineRunInterface.List(metav1.ListOptions{})
	if err != nil {
		log.Logger().Warnf("no PipelineRun instances found: %s", err.Error())
		return nil
	}

	now := time.Now()
	counters := &buildsCount{}

	// lets go in reverse order so we delete the oldest first
	for i := len(activities.Items) - 1; i >= 0; i-- {
		a := activities.Items[i]
		var isPR, isBatch bool
		if a.Labels != nil {
			isPR, isBatch = o.isPullRequestOrBatchBranch(a.Labels["branch"])
		}
		maxAge, revisionHistory := o.ageAndHistoryLimits(isPR, isBatch)

		completionTime := a.Status.CompletionTime
		if completionTime != nil && completionTime.Add(maxAge).Before(now) {
			err = o.deletePipelineRun(pipelineRunInterface, &a)
			if err != nil {
				return err
			}
			continue
		}

		labels := a.Labels
		if labels != nil {
			owner := labels[v1.LabelOwner]
			repo := labels[v1.LabelRepository]
			if repo == "" {
				repo = labels["repo"]
			}
			branch := labels[v1.LabelBranch]

			// TODO another way to uniquely find the git repo + branch?
			if owner != "" && repo != "" && branch != "" {
				repoAndBranchName := owner + "/" + repo + "/" + branch
				c := counters.AddBuild(repoAndBranchName, isPR)
				if c > revisionHistory {
					err = o.deletePipelineRun(pipelineRunInterface, &a)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (o *GCActivitiesOptions) deleteActivity(activityInterface jv1.PipelineActivityInterface, a *v1.PipelineActivity) error {
	prefix := ""
	if o.DryRun {
		prefix = "not "
	}
	log.Logger().Infof("%sdeleting PipelineActivity %s", prefix, util.ColorInfo(a.Name))
	if o.DryRun {
		return nil
	}
	return activityInterface.Delete(a.Name, metav1.NewDeleteOptions(0))
}

func (o *GCActivitiesOptions) deletePipelineRun(pipelineRunInterface tv1alpha1.PipelineRunInterface, a *v1alpha1.PipelineRun) error {
	prefix := ""
	if o.DryRun {
		prefix = "not "
	}
	log.Logger().Infof("%sdeleting PipelineRun %s", prefix, util.ColorInfo(a.Name))
	if o.DryRun {
		return nil
	}
	return pipelineRunInterface.Delete(a.Name, metav1.NewDeleteOptions(0))
}

func (o *GCActivitiesOptions) ageAndHistoryLimits(isPR, isBatch bool) (time.Duration, int) {
	maxAge := o.ReleaseAgeLimit
	revisionLimit := o.ReleaseHistoryLimit
	if isPR || isBatch {
		maxAge = o.PullRequestAgeLimit
		revisionLimit = o.PullRequestHistoryLimit
	}
	return maxAge, revisionLimit
}

func (o *GCActivitiesOptions) isPullRequestOrBatchBranch(branchName string) (bool, bool) {
	return strings.HasPrefix(branchName, "PR-"), branchName == "batch"
}
