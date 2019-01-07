package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/workflow"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetWorkflowOptions containers the CLI options
type GetWorkflowOptions struct {
	GetOptions

	Name string
}

var (
	getWorkflowLong = templates.LongDesc(`
		Display either all the workflows or a specific workflow
`)

	getWorkflowExample = templates.Examples(`
		# List all the available workflows
		jx get workflow

		# Display a specific workflow
		jx get workflow -n default
	`)
)

// NewCmdGetWorkflow creates the new command for: jx get env
func NewCmdGetWorkflow(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetWorkflowOptions{
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
		Use:     "workflows",
		Short:   "Display either all the available workflows or a specific workflow",
		Aliases: []string{"workflow"},
		Long:    getWorkflowLong,
		Example: getWorkflowExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "The name of the workflow to display")

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetWorkflowOptions) Run() error {
	jxClient, ns, err := o.JXClientAndAdminNamespace()
	if err != nil {
		return err
	}

	name := o.Name
	if name == "" {
		if len(o.Args) > 0 {
			name = o.Args[0]
		}
	}

	if name != "" {
		return o.getWorkflow(name, jxClient, ns)
	}

	workflows, err := jxClient.JenkinsV1().Workflows(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	table := o.createTable()
	table.AddRow("WORKFLOW")
	for _, workflow := range workflows.Items {
		table.AddRow(workflow.Name)
	}
	table.Render()
	return nil
}

func (o *GetWorkflowOptions) getWorkflow(name string, jxClient versioned.Interface, ns string) error {
	workflow, err := workflow.GetWorkflow(name, jxClient, ns)
	if err != nil {
		return err
	}

	log.Infof("Workflow: %s\n", workflow.Name)
	lines := []*StepSummary{}
	var lastSummary *StepSummary
	for _, step := range workflow.Spec.Steps {
		promote := step.Promote
		if promote != nil {
			if len(step.Preconditions.Environments) > 0 {
				lastSummary = nil
			}
			if lastSummary == nil {
				lastSummary = &StepSummary{
					Action: "promote",
					// Resources: []string{},
				}
				lines = append(lines, lastSummary)
			}
			lastSummary.Resources = append(lastSummary.Resources, promote.Environment)
			if len(step.Preconditions.Environments) > 0 {
				lastSummary = nil
			}
		}
	}
	for i, summary := range lines {
		if i > 0 {
			log.Info("    |\n")
		}
		log.Infof("%s to %s\n", summary.Action, strings.Join(summary.Resources, " + "))
	}
	return nil
}

type StepSummary struct {
	Action    string
	Resources []string
}
