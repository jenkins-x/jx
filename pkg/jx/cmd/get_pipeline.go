package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
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
		Short:   "Display one or many pipelines",
		Long:    get_pipeline_long,
		Example: get_pipeline_example,
		Aliases: []string{ "pipe", "pipes", "pipeline"},
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
	table.AddRow("Name", "URL")

	for _, job := range jobs {
		table.AddRow(job.Name, job.Url)
	}
	table.Render()
	return nil
}
