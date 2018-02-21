package cmd

import (
	"github.com/spf13/cobra"
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetActivityOptions containers the CLI options
type GetActivityOptions struct {
	CommonOptions
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
		CommonOptions{
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
	table.AddRow("STEP", "STATUS", "STARTED", "DURATION", "DESCRIPTION")

	for _, activity := range list.Items {
		if o.matches(&activity) {
			spec := &activity.Spec
			table.AddRow("Pipeline "+spec.Pipeline+" #"+spec.Build,
				spec.Status.String(),
				timeToString(spec.StartedTimestamp),
				durationString(spec.StartedTimestamp, spec.CompletedTimestamp))

			indent := "  "
			for _, step := range spec.Steps {
				name, description := stepText(&step)
				table.AddRow(indent+name,
					spec.Status.String(),
					timeToString(spec.StartedTimestamp),
					durationString(spec.StartedTimestamp, spec.CompletedTimestamp),
					description)
			}
		}
	}

	/*
		for _, ea := range envApps {
			titles = append(titles, strings.ToUpper(ea.Environment.Name), "PODS")
		}
	*/

	table.Render()
	return nil
}

func stepText(parent *v1.PipelineActivityStep) (string, string) {
	stage := parent.Stage
	step := parent.Step
	promotePullRequest := parent.PromotePullRequest
	promote := parent.Promote
	if stage != nil {
		return "Stage " + stage.Name, stage.Description
	} else if step != nil {
		return "Step " + step.Name, step.Description
	} else if promotePullRequest != nil {
		return "PromotePullRequest " + promotePullRequest.Name, promotePullRequest.Description
	} else if promote != nil {
		return "Promote " + promote.Name, promote.Description
	}
	return "Unknown", ""
}

func durationString(start *metav1.Time, end *metav1.Time) string {
	if start == nil || end == nil {
		return ""
	}
	return end.Sub(start.Time).String()
}

func timeToString(time *metav1.Time) string {
	if time == nil {
		return ""
	}
	return time.String()
}

func (o *GetActivityOptions) matches(activity *v1.PipelineActivity) bool {
	return true
}
