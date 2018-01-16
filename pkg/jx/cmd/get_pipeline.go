package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
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

	return cmd
}

// Run implements this command
func (o *GetPipelineOptions) Run() error {
	f := o.Factory
	jenkins, err := f.GetJenkinsClient()
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

	table := o.CreateTable()
	table.AddRow("Name", "URL", "lastBuild", "status", "duration")

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
				table.AddRow(job.FullName, job.Url, "", "Never Built", "")
			}
			return nil
		}
		if last.Building {
			table.AddRow(job.FullName, job.Url, "#"+last.Id, "Building", time.Duration(last.EstimatedDuration).String()+"(est.)")
		} else {
			table.AddRow(job.FullName, job.Url, "#"+last.Id, last.Result, time.Duration(last.Duration).String())
		}
	}
	return nil
}
