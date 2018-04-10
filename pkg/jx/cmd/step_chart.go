package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/issues"
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

	FromDate             string
	ToDate               string
	Dir                  string
	BlogOutputDir        string
	BlogName             string
	CombineMinorReleases bool

	State StepChartState
}

type StepChartState struct {
	GitInfo     *gits.GitRepositoryInfo
	GitProvider gits.GitProvider
	Tracker     issues.IssueProvider
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
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to query to find the projects .git directory")
	cmd.Flags().StringVarP(&options.FromDate, "from-date", "f", "", "The date to create the charts from. Defaults to a week before the to date")
	cmd.Flags().StringVarP(&options.ToDate, "to-date", "t", "", "The date to query to")
	cmd.Flags().StringVarP(&options.BlogOutputDir, "blog-dir", "", "", "The Hugo-style blog source code to generate the charts into")
	cmd.Flags().StringVarP(&options.BlogName, "blog-name", "n", "", "The blog name")
	cmd.Flags().BoolVarP(&options.CombineMinorReleases, "combine-minor", "c", true, "If enabled lets combine minor releases together to simplify the charts")
	return cmd
}

// Run implements this command
func (o *StepChartOptions) Run() error {
	outDir := o.BlogOutputDir
	if outDir != "" {
		if o.BlogName == "" {
			t := time.Now()
			o.BlogName = "changes-" + strconv.Itoa(t.Day()) + "-" + strings.ToLower(t.Month().String()) + "-" + strconv.Itoa(t.Year())
		}
		state, err := o.generateChangelog()
		if err != nil {
			return err
		}
		o.State = state
	} else {
		gitInfo, gitProvider, tracker, err := o.createGitProvider(o.Dir)
		if err != nil {
			return err
		}
		if gitInfo == nil {
			return fmt.Errorf("Could not find a .git folder in the current directory so could not determine the current project")
		}
		o.State = StepChartState{
			GitInfo:     gitInfo,
			GitProvider: gitProvider,
			Tracker:     tracker,
		}
	}
	return o.downloadsReport(o.State.GitProvider, o.State.GitInfo.Organisation, o.State.GitInfo.Name)
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
	if o.CombineMinorReleases {
		releases = o.combineMinorReleases(releases)
	}
	o.Printf("Found %d releases\n", len(releases))

	report := o.createBarReport("downloads", "Version", "Downloads")

	for _, release := range releases {
		report.AddNumber(release.Name, release.DownloadCount)
	}
	report.Render()
	return nil
}

// createBarReport creates the new report instance
func (o *StepChartOptions) createBarReport(name string, legends ...string) reports.BarReport {
	outDir := o.BlogOutputDir
	if outDir != "" {
		blogName := o.BlogName
		if blogName == "" {
			t := time.Now()
			blogName = fmt.Sprintf("changes-%d-%s-%d", t.Day(), t.Month().String(), t.Year())
		}

		jsFileName := filepath.Join(outDir, "static", "news", blogName, name+".js")
		jsLinkURI := filepath.Join("/news", blogName, name+".js")
		/*		var buffer bytes.Buffer
				writer := bufio.NewWriter(&buffer)
				writer.WriteString("\n### " + name + "\n\n")
				writer.Flush()
				o.Printf(buffer.String())
		*/
		return reports.NewBlogBarReport(name, o.Out, jsFileName, jsLinkURI)
	}
	return reports.NewTableBarReport(o.CreateTable(), legends...)
}

func (options *StepChartOptions) combineMinorReleases(releases []*gits.GitRelease) []*gits.GitRelease {
	answer := []*gits.GitRelease{}
	m := map[string]*gits.GitRelease{}
	for _, release := range releases {
		name := release.Name
		if name != "" {
			idx := strings.LastIndex(name, ".")
			if idx > 0 {
				name = name[0:idx] + ".x"
			}
		}
		cur := m[name]
		if cur == nil {
			copy := *release
			copy.Name = name
			m[name] = &copy
			answer = append(answer, &copy)
		} else {
			cur.DownloadCount += release.DownloadCount
		}
	}
	return answer
}

func (o *StepChartOptions) generateChangelog() (StepChartState, error) {
	blogFile := filepath.Join(o.BlogOutputDir, "content", "news", o.BlogName+".md")
	previousDate := o.FromDate
	if previousDate == "" {
		// default to 4 weeks ago
		t := time.Now().Add(-time.Hour * 24 * 7 * 4)
		previousDate = gits.FormatDate(t)
	}
	options := &StepChangelogOptions{
		StepOptions: o.StepOptions,
		Dir:         o.Dir,
		Version:     "Changes",
		// TODO add time now and previous time
		PreviousDate:       previousDate,
		OutputMarkdownFile: blogFile,
	}
	err := options.Run()
	answer := StepChartState{}
	if err != nil {
		return answer, err
	}
	state := &options.State
	answer.GitInfo = state.GitInfo
	answer.GitProvider = state.GitProvider
	answer.Tracker = state.Tracker
	return answer, nil
}
