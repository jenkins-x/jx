package get

import (
	"fmt"
	"sort"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/tekton"
	"github.com/pkg/errors"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/v2/pkg/prow"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	table2 "github.com/jenkins-x/jx/v2/pkg/table"
)

// PipelineOptions is the start of the data required to perform the operation.
// As new fields are added, add them here instead of
// referencing the cmd.Flags()
type PipelineOptions struct {
	Options
	ProwOptions prow.Options
}

var (
	getPipelineLong = templates.LongDesc(`
		Display one or more pipelines.

`)

	getPipelineExample = templates.Examples(`
		# list all pipelines
		jx get pipeline
	`)
)

// NewCmdGetPipeline creates the command
func NewCmdGetPipeline(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &PipelineOptions{
		Options: Options{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "pipelines [flags]",
		Short:   "Display one or more Pipelines",
		Long:    getPipelineLong,
		Example: getPipelineExample,
		Aliases: []string{"pipe", "pipes", "pipeline"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.AddGetFlags(cmd)

	return cmd
}

// Run implements this command
func (o *PipelineOptions) Run() error {
	tektonClient, ns, err := o.TektonClient()
	if err != nil {
		return errors.Wrap(err, "could not create tekton client")
	}

	pipelines := tektonClient.TektonV1alpha1().PipelineRuns(ns)
	prList, err := pipelines.List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to list PipelineRuns in namespace %s", ns)
	}

	if len(prList.Items) == 0 {
		return errors.New(fmt.Sprintf("no PipelineRuns were found in namespace %s", ns))
	}

	var owner, repo, branch, context, buildNumber, status string
	names := []string{}
	m := map[string]*pipelineapi.PipelineRun{}
	for k := range prList.Items {
		pr := prList.Items[k]
		status = "not completed"
		if tekton.PipelineRunIsComplete(&pr) {
			status = "completed"
		}
		labels := pr.Labels
		if labels == nil {
			continue
		}
		owner = labels[tekton.LabelOwner]
		repo = labels[tekton.LabelRepo]
		branch = labels[tekton.LabelBranch]
		context = labels[tekton.LabelContext]
		buildNumber = labels[tekton.LabelBuild]

		if owner == "" {
			log.Logger().Warnf("missing label %s on PipelineRun %s has labels %#v", tekton.LabelOwner, pr.Name, labels)
			continue
		}
		if repo == "" {
			log.Logger().Warnf("missing label %s on PipelineRun %s has labels %#v", tekton.LabelRepo, pr.Name, labels)
			continue
		}
		if branch == "" {
			log.Logger().Warnf("missing label %s on PipelineRun %s has labels %#v", tekton.LabelBranch, pr.Name, labels)
			continue
		}

		name := fmt.Sprintf("%s/%s/%s #%s %s", owner, repo, branch, buildNumber, status)

		if context != "" {
			name = fmt.Sprintf("%s-%s", name, context)
		}
		names = append(names, name)
		m[name] = &pr
	}

	sort.Strings(names)

	if o.Output != "" {
		return o.renderResult(names, o.Output)
	}

	table := createTable(o)

	for _, j := range names {
		table.AddRow(j, "N/A", "N/A", "N/A", "N/A")
	}
	table.Render()

	return nil
}

func createTable(o *PipelineOptions) table2.Table {
	table := o.CreateTable()
	table.AddRow("Name", "URL", "LAST_BUILD", "STATUS", "DURATION")
	return table
}
