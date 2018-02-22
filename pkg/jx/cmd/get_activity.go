package cmd

import (
	"io"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	tbl "github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// GetActivityOptions containers the CLI options
type GetActivityOptions struct {
	CommonOptions

	Filter      string
	BuildNumber string
}

var (
	get_activity_long = templates.LongDesc(`
		Display the current activities for one more more projects.
`)

	get_activity_example = templates.Examples(`
		# List the current activities for all applications in the current team
		jx get activities

		# List the current activities for application 'foo'
		jx get act foo
	`)
)

// NewCmdGetActivity creates the new command for: jx get version
func NewCmdGetActivity(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetActivityOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "activities",
		Short:   "Display one or many Activities on projects",
		Aliases: []string{"activity", "act"},
		Long:    get_activity_long,
		Example: get_activity_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Text to filter the pipeline names")
	cmd.Flags().StringVarP(&options.BuildNumber, "build", "b", "", "The build number to filter on")
	return cmd
}

// Run implements this command
func (o *GetActivityOptions) Run() error {
	f := o.Factory
	client, currentNs, err := f.CreateJXClient()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}
	envList, err := client.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	kube.SortEnvironments(envList.Items)

	apisClient, err := f.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	list, err := client.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	table := o.CreateTable()
	table.SetColumnAlign(1, tbl.ALIGN_RIGHT)
	table.SetColumnAlign(2, tbl.ALIGN_RIGHT)
	table.AddRow("STEP", "STARTED AGO", "DURATION", "STATUS")

	for _, activity := range list.Items {
		if o.matches(&activity) {
			spec := &activity.Spec
			text := ""
			version := activity.Spec.Version
			if version != "" {
				text = "Version: " + util.ColorInfo(version)
			}
			statusText := statusString(activity.Spec.Status)
			if statusText == "" {
				statusText = text
			} else {
				statusText += " " + text
			}
			table.AddRow(spec.Pipeline+" #"+spec.Build,
				timeToString(spec.StartedTimestamp),
				durationString(spec.StartedTimestamp, spec.CompletedTimestamp),
				statusText)

			indent := "  "
			for _, step := range spec.Steps {
				o.addStepRow(&table, &step, indent)
			}
		}
	}
	table.Render()
	return nil
}

func (o *CommonOptions) addStepRow(table *tbl.Table, parent *v1.PipelineActivityStep, indent string) {
	stage := parent.Stage
	step := parent.Step
	promotePullRequest := parent.PromotePullRequest
	promote := parent.Promote
	if stage != nil {
		addStepRowItem(table, stage, indent, "Stage", "")
	} else if step != nil {
		addStepRowItem(table, step, indent, "", "")
	} else if promotePullRequest != nil {
		addStepRowItem(table, &promotePullRequest.CoreActivityStep, indent, "PromotePullRequest: "+promotePullRequest.Environment, describePullRequest(promotePullRequest))
	} else if promote != nil {
		addStepRowItem(table, &promote.CoreActivityStep, indent, "Promote: "+promote.Environment, describePullRequest(promote))
	} else {
		o.warnf("Unknown step kind %#v\n", parent)
	}
}

func addStepRowItem(table *tbl.Table, step *v1.CoreActivityStep, indent string, name string, description string) {
	text := step.Description
	if description != "" {
		if text == "" {
			text = description
		} else {
			text += description
		}
	}
	textName := step.Name
	if textName == "" {
		textName = name
	} else {
		if name != "" {
			textName = name + ":" + textName
		}
	}
	table.AddRow(indent+textName,
		timeToString(step.StartedTimestamp),
		durationString(step.StartedTimestamp, step.CompletedTimestamp),
		statusString(step.Status)+text)
}

func statusString(statusType v1.ActivityStatusType) string {
	text := statusType.String()
	switch statusType {
	case v1.ActivityStatusTypeFailed, v1.ActivityStatusTypeError:
		return util.ColorError(text)
	case v1.ActivityStatusTypeSucceeded:
		return util.ColorInfo(text)
	case v1.ActivityStatusTypeRunning:
		return util.ColorStatus(text)
	}
	return text
}

func describePullRequest(promotePullRequest *v1.PromotePullRequestStep) string {
	description := ""
	if promotePullRequest.PullRequestURL != "" {
		description += " PullRequest: " + util.ColorInfo(promotePullRequest.PullRequestURL)
	}
	if promotePullRequest.MergeCommitSHA != "" {
		description += " Merge SHA: " + util.ColorInfo(promotePullRequest.MergeCommitSHA)
	}
	return description
}

func durationString(start *metav1.Time, end *metav1.Time) string {
	if start == nil || end == nil {
		return ""
	}
	return end.Sub(start.Time).Round(time.Second).String()
}

func timeToString(t *metav1.Time) string {
	if t == nil {
		return ""
	}
	now := &metav1.Time{
		Time: time.Now(),
	}
	return durationString(t, now)
}

func (o *GetActivityOptions) matches(activity *v1.PipelineActivity) bool {
	answer := true
	filter := o.Filter
	if filter != "" {
		answer = strings.Contains(activity.Name, filter) || strings.Contains(activity.Spec.Pipeline, filter)
	}
	build := o.BuildNumber
	if answer && build != "" {
		answer = activity.Spec.Build == build
	}
	return answer
}
