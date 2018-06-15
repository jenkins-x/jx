package cmd

import (
	"io"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"strconv"

	"sort"

	"fmt"

	"strings"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GCActivitiesOptions struct {
	CommonOptions

	RevisionHistoryLimit int
	jclient              *gojenkins.Jenkins
}

var (
	GCActivitiesLong = templates.LongDesc(`
		Garbage collect the Jenkins X Activity Custom Resource Definitions

`)

	GCActivitiesExample = templates.Examples(`
		jx garbage collect activities
		jx gc activities
`)
)

// NewCmd s a command object for the "step" command
func NewCmdGCActivities(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GCActivitiesOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "activities",
		Short:   "garbage collection for activities",
		Long:    GCActivitiesLong,
		Example: GCActivitiesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().IntVarP(&options.RevisionHistoryLimit, "revision-history-limit", "", 5, "Minimum number of Activities per application to keep")
	return cmd
}

// Run implements this command
func (o *GCActivitiesOptions) Run() error {
	f := o.Factory
	client, currentNs, err := f.CreateJXClient()
	if err != nil {
		return err
	}

	// cannot use field selectors like `spec.kind=Preview` on CRDs so list all environments
	activities, err := client.JenkinsV1().PipelineActivities(currentNs).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(activities.Items) == 0 {
		// no preview environments found so lets return gracefully
		if o.Verbose {
			log.Info("no activities found\n")
		}
		return nil
	}

	o.jclient, err = o.Factory.CreateJenkinsClient()
	if err != nil {
		return err
	}

	jobs, err := o.jclient.GetJobs()
	if err != nil {
		return err
	}
	var jobNames []string
	for _, j := range jobs {
		err = o.getAllPipelineJobNames(&jobNames, j.Name)
		if err != nil {
			return err
		}
	}

	activityBuilds := make(map[string][]int)

	for _, a := range activities.Items {
		// if activity has no job in jenkins delete it
		matched := false
		for _, j := range jobNames {
			if a.Spec.Pipeline == j {
				matched = true
				break
			}
		}
		if !matched {
			err = client.JenkinsV1().PipelineActivities(currentNs).Delete(a.Name, metav1.NewDeleteOptions(0))
			if err != nil {
				return err
			}
		}

		buildNumber, err := strconv.Atoi(a.Spec.Build)
		if err != nil {
			return err
		}
		// collect all activities for a pipeline
		activityBuilds[a.Spec.Pipeline] = append(activityBuilds[a.Spec.Pipeline], buildNumber)
	}

	for pipeline, builds := range activityBuilds {

		sort.Ints(builds)

		// iterate over the build numbers and delete any while the activity is under the RevisionHistoryLimit
		i := 0
		for i < len(builds)-o.RevisionHistoryLimit {
			activityName := fmt.Sprintf("%s-%v", pipeline, builds[i])
			activityName = strings.Replace(activityName, "/", "-", -1)
			activityName = strings.ToLower(activityName)

			err = client.JenkinsV1().PipelineActivities(currentNs).Delete(activityName, metav1.NewDeleteOptions(0))
			if err != nil {
				return fmt.Errorf("failed to delete activity %s: %v\n", activityName, err)
			}

			i++
		}
	}

	return nil
}
func (o *GCActivitiesOptions) getAllPipelineJobNames(jobNames *[]string, jobName string) error {

	job, err := o.jclient.GetJob(jobName)
	if err != nil {
		return err
	}

	if len(job.Jobs) == 0 {

		*jobNames = append(*jobNames, job.FullName)
	}

	for _, j := range job.Jobs {
		err = o.getAllPipelineJobNames(jobNames, job.FullName+"/"+j.Name)
		if err != nil {
			return err
		}
	}
	return nil
}
