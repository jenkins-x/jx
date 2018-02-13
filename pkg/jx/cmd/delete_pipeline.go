package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"fmt"
)

// DeleteJenkinsOptions are the flags for delete commands
type DeletePipelineOptions struct {
	CommonOptions
}

var (
	delete_pipeline_long = templates.LongDesc(`
		Delete one or many pipeline jobs.

`)

	delete_pipeline_example = templates.Examples(`
		# Delete one or many pipeline jobs.
		jx delete pipeline foo [bar]
	`)
)


// NewCmdDeletePipeline creates a command object for the generic "post" action to delete a jenkins job
func NewCmdDeletePipeline(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeletePipelineOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Delete one or many pipeline jobs",
		Long: delete_pipeline_long,
		Example: delete_pipeline_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm"},
	}
	return cmd
}

// Run implements this command
func (o *DeletePipelineOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return fmt.Errorf("Missing pipleline name argument")
	}

	f := o.Factory
	jenkins, err := f.CreateJenkinsClient()
	if err != nil {
		return err
	}
	for _, name := range args {
		job, err := jenkins.GetJob(name)
		if err != nil {
			return err
		}
		jenkins.DeleteJob(job)
	}
	return nil
}
