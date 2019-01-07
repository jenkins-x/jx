package cmd

import (
	"errors"
	"io"
	"sort"

	"github.com/jenkins-x/jx/pkg/prow"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

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

	ProwOptions prow.Options
}

var (
	get_pipeline_long = templates.LongDesc(`
		Display one or more pipelines.

`)

	get_pipeline_example = templates.Examples(`
		# List all pipelines
		jx get pipeline
	`)
)

// NewCmdGetPipeline creates the command
func NewCmdGetPipeline(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetPipelineOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "pipelines [flags]",
		Short:   "Display one or more Pipelines",
		Long:    get_pipeline_long,
		Example: get_pipeline_example,
		Aliases: []string{"pipe", "pipes", "pipeline"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetPipelineOptions) Run() error {

	_, _, err := o.JXClient()
	if err != nil {
		return err
	}

	_, _, err = o.KubeClient()
	if err != nil {
		return err
	}

	isProw, err := o.isProw()
	if err != nil {
		return err
	}

	if isProw {
		o.ProwOptions = prow.Options{
			KubeClient: o.KubeClientCached,
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

	} else {
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

		table := createTable(o)

		for _, j := range jobs {
			job, err := jenkins.GetJob(j.Name)
			if err != nil {
				return err
			}
			o.dump(jenkins, job.Name, &table)
		}
		table.Render()
	}
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
