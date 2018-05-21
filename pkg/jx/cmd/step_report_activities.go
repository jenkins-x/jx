package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// StepReportActivitiesOptions contains the command line flags
type StepReportActivitiesOptions struct {
	StepReportOptions
	Watch bool
}

var (
	StepReportActivitiesLong = templates.LongDesc(`
		This pipeline step reports activities to pluggable backends like ElasticSearch
`)

	StepReportActivitiesExample = templates.Examples(`
		jx step report activities
`)
)

func NewCmdStepReportActivities(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepReportActivitiesOptions{
		StepReportOptions: StepReportOptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "activities",
		Short:   "Reports activities",
		Long:    StepReportActivitiesLong,
		Example: StepReportActivitiesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Watch, "watch", "w", false, "Whether to watch activities")
	return cmd
}

func (o *StepReportActivitiesOptions) Run() error {

	return nil
}
