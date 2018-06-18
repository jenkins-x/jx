package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"strings"
	"time"
)

// GetPipelineOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type GetPipelineOptions struct {
	GetOptions
}

var (
	get_pipeline_long = templates.LongDesc(`
		Display one or many pipelines.

`)

	get_pipeline_example = templates.Examples(`
		# List all pipelines
		jx get pipeline
	`)
)

// NewCmdGetPipeline creates the command
func NewCmdGetPipeline(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetPipelineOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "pipelines [flags]",
		Short:   "Display one or many Pipelines",
		Long:    get_pipeline_long,
		Example: get_pipeline_example,
		Aliases: []string{"pipe", "pipes", "pipeline"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetPipelineOptions) Run() error {
	jenkins, err := o.JenkinsClient()
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

	table := o.CreateTable()
	table.AddRow("Name", "URL", "LAST_BUILD", "STATUS", "DURATION")

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

func (o *GetPipelineOptions) dump(jenkins *gojenkins.Jenkins, name string, table *table.Table) error {
	job, err := jenkins.GetJob(name)
	if err != nil {
		return err
	}

	if job.Jobs != nil {
		for _, child := range job.Jobs {
			o.dump(jenkins, job.FullName+"/"+child.Name, table)
		}
	} else {
		last, err := jenkins.GetLastBuild(job)
		if err != nil {
			if jenkins.IsErrNotFound(err) {
				if o.matchesFilter(&job) {
					table.AddRow(job.FullName, job.Url, "", "Never Built", "")
				}
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
