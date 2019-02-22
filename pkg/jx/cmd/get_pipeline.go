package cmd

import (
	"errors"
	"github.com/jenkins-x/jx/pkg/log"
	"net/url"
	"sort"

	"github.com/jenkins-x/jx/pkg/prow"

	"github.com/spf13/cobra"

	"strings"
	"time"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/table"
)

// GetPipelineOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GetPipelineOptions struct {
	GetOptions

	JenkinsSelector JenkinsSelectorOptions

	ProwOptions prow.Options
}

var (
	getPipelineLong = templates.LongDesc(`
		Display one or more pipelines.

`)

	getPipelineExample = templates.Examples(`
		# list all pipelines
		jx get pipeline

		# Lists all the pipelines in a custom Jenkins App
		jx get pipeline -c
	`)
)

// NewCmdGetPipeline creates the command
func NewCmdGetPipeline(commonOpts *CommonOptions) *cobra.Command {
	options := &GetPipelineOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "pipelines [flags]",
		Short:   "Display one or more Pipelines",
		Long:    getPipelineLong,
		Example: getPipelineExample,
		Aliases: []string{"pipe", "pipes", "pipeline"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addGetFlags(cmd)

	cmd.Flags().BoolVarP(&options.JenkinsSelector.UseCustomJenkins, "custom", "c", false, "List the pipelines in custom Jenkins App instead of the default execution engine in Jenkins X")
	cmd.Flags().StringVarP(&options.JenkinsSelector.CustomJenkinsName, "name", "n", "", "The name of the custom Jenkins App if you don't wish to list the pipelines in the default execution engine in Jenkins X")

	return cmd
}

// Run implements this command
func (o *GetPipelineOptions) Run() error {
	jo := &o.JenkinsSelector
	if jo.CustomJenkinsName != "" {
		jo.UseCustomJenkins = true
	}

	_, _, err := o.JXClient()
	if err != nil {
		return err
	}

	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	isProw, err := o.isProw()
	if err != nil {
		return err
	}

	if jo.UseCustomJenkins || !isProw {
		jenkins, err := o.CreateCustomJenkinsClient(jo)
		if err != nil {
			return err
		}
		jobs, err := jenkins.GetJobs()
		if err != nil {
			return err
		}
		if len(jobs) == 0 {
			return outputEmptyListWarning(o.Out)
		}

		if o.Output != "" {
			return o.renderResult(jobs, o.Output)
		}

		table := createTable(o)

		for _, j := range jobs {
			job, err := jenkins.GetJob(j.Name)
			if err != nil {
				return err
			}
			o.dump(jenkins, job.Name, &table)
		}
		table.Render()
		return nil
	}
	o.ProwOptions = prow.Options{
		KubeClient: client,
		NS:         o.currentNamespace,
	}
	names, err := o.ProwOptions.GetReleaseJobs()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return errors.New("no pipelines found")
	}
	sort.Strings(names)

	if len(names) == 0 {
		return outputEmptyListWarning(o.Out)
	}

	if o.Output != "" {
		return o.renderResult(names, o.Output)
	}

	table := createTable(o)

	for _, j := range names {
		if err != nil {
			return err
		}
		table.AddRow(j, "N/A", "N/A", "N/A", "N/A")
	}
	table.Render()

	return nil
}

func createTable(o *GetPipelineOptions) table.Table {
	table := o.createTable()
	table.AddRow("Name", "URL", "LAST_BUILD", "STATUS", "DURATION")
	return table
}

func (o *GetPipelineOptions) dump(jenkins gojenkins.JenkinsClient, name string, table *table.Table) error {
	job, err := jenkins.GetJob(name)
	if err != nil {
		return err
	}

	if job.Jobs != nil {
		for _, child := range job.Jobs {
			o.dump(jenkins, job.FullName+"/"+child.Name, table)
		}
		if len(job.Jobs) == 0 {
			log.Warnf("Job %s has no children!\n", job.Name)
		}
	} else {
		job.Url = switchJenkinsBaseURL(job.Url, jenkins.BaseURL())
		last, err := jenkins.GetLastBuild(job)

		if err != nil {
			if jenkins.IsErrNotFound(err) {
				if o.matchesFilter(&job) {
					table.AddRow(job.FullName, job.Url, "", "Never Built", "")
				}
			} else {
				log.Warnf("Failed to find last build for job %s: %s\n", job.Name, err.Error())
			}
			return nil
		}
		if o.matchesFilter(&job) {
			if last.Building {
				table.AddRow(job.FullName, job.Url, "#"+last.Id, "Building", time.Duration(last.EstimatedDuration).String()+"(est.)")
			} else {
				table.AddRow(job.FullName, job.Url, "#"+last.Id, last.Result, time.Duration(last.Duration).String())
			}
		}
	}
	return nil
}

// switchJenkinsBaseURL sometimes a Jenkins server does not know its external URL so lets switch the base URL of the job
// URL to use the known working baseURL of the jenkins server
func switchJenkinsBaseURL(jobURL string, baseURL string) string {
	if jobURL == "" {
		return baseURL
	}
	if baseURL == "" {
		return jobURL
	}
	u, err := url.Parse(jobURL)
	if err != nil {
		log.Warnf("failed to parse Jenkins Job URL %s due to: %s\n", jobURL, err)
		return jobURL
	}

	u2, err := url.Parse(baseURL)
	if err != nil {
		log.Warnf("failed to parse Jenkins base URL %s due to: %s\n", baseURL, err)
		return jobURL
	}
	u.Host = u2.Host
	u.Scheme = u2.Scheme
	return u.String()
}

func (o *GetPipelineOptions) matchesFilter(job *gojenkins.Job) bool {
	args := o.Args
	if len(args) == 0 {
		return true
	}
	name := job.FullName
	for _, arg := range args {
		if strings.Contains(name, arg) {
			return true
		}
	}
	return false
}
