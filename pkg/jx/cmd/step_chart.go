package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/reports"
	"github.com/spf13/cobra"
)

const ()

var (
	stepChartLong = templates.LongDesc(`
		Generates charts for a project
`)

	stepChartExample = templates.Examples(`
		# create charts for the cuect
		jx step chart

			`)
)

// StepChartOptions contains the command line flags
type StepChartOptions struct {
	StepOptions

	FromDate string
	ToDate   string
	Dir      string

	State StepChartState
}

type StepChartState struct {
	GitInfo     *gits.GitRepositoryInfo
	GitProvider gits.GitProvider
}

// NewCmdStepChart Creates a new Command object
func NewCmdStepChart(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepChartOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "chart",
		Short:   "Creates charts for project metrics",
		Long:    stepChartLong,
		Example: stepChartExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to query to find the projects .git directory")
	cmd.Flags().StringVarP(&options.FromDate, "from-date", "f", "", "The date to create the charts from. Defaults to a week before the to date")
	cmd.Flags().StringVarP(&options.ToDate, "to-date", "t", "", "The date to query to")
	return cmd
}

// Run implements this command
func (o *StepChartOptions) Run() error {
	gitInfo, gitProvider, err := o.createGitProvider(o.Dir)
	if err != nil {
		return err
	}
	if gitInfo == nil {
		return fmt.Errorf("Could not find a .git folder in the current directory so could not determine the current project")
	}
	o.State = StepChartState{
		GitInfo:     gitInfo,
		GitProvider: gitProvider,
	}
	return o.downloadsReport(gitProvider, gitInfo.Organisation, gitInfo.Name)
}

func (o *StepChartOptions) downloadsReport(provider gits.GitProvider, owner string, repo string) error {
	releases, err := provider.ListReleases(owner, repo)
	if err != nil {
		return err
	}
	if len(releases) == 0 {
		o.warnf("No releases found for %s/%s/n", owner, repo)
		return nil
	}
	o.Printf("Found %d releases\n", len(releases))

	report := reports.NewTableBarReport(o.CreateTable())

	for _, release := range releases {
		report.AddNumber(release.Name, release.DownloadCount)
	}
	report.Render()
	return nil
}
