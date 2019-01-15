package cmd

import (
	"io"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
)

// GetDevPodOptions the command line options
type GetDevPodOptions struct {
	GetOptions
	CommonDevPodOptions
}

var (
	getDevPodLong = templates.LongDesc(`
		Display the available DevPods

		For more documentation see: [https://jenkins-x.io/developing/devpods/](https://jenkins-x.io/developing/devpods/)

`)

	getDevPodExample = templates.Examples(`
		# List all the possible DevPods
		jx get devPod
	`)
)

// NewCmdGetDevPod creates the command
func NewCmdGetDevPod(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetDevPodOptions{
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
		Use:     "devpod [flags]",
		Short:   "Lists the DevPods",
		Long:    getDevPodLong,
		Example: getDevPodExample,
		Aliases: []string{"buildpod", "buildpods", "devpods"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonDevPodFlags(cmd)

	return cmd
}

// Run implements this command
func (o *GetDevPodOptions) Run() error {

	client, curNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(client, curNs)
	if err != nil {
		return err
	}

	userName, err := o.getUsername(o.Username)
	if err != nil {
		return err
	}

	names, m, err := kube.GetDevPodNames(client, ns, userName)

	table := o.createTable()
	table.AddRow("NAME", "POD TEMPLATE", "AGE", "STATUS")

	for _, k := range names {
		pod := m[k]
		if pod != nil {
			podTemplate := ""
			status := kube.PodStatus(pod)
			labels := pod.Labels
			d := time.Now().Sub(pod.CreationTimestamp.Time).Round(time.Second)
			age := d.String()
			if labels != nil {
				podTemplate = labels[kube.LabelPodTemplate]
			}
			table.AddRow(k, podTemplate, age, status)
		}
	}

	table.Render()
	return nil
}
