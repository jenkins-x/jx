package get

import (
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
)

// GetDevPodOptions the command line options
type GetDevPodOptions struct {
	GetOptions
	opts.CommonDevPodOptions

	AllUsernames bool
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
func NewCmdGetDevPod(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetDevPodOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
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
			helper.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.AllUsernames, "all-usernames", "", false, "Gets devpods for all usernames")

	options.AddCommonDevPodFlags(cmd)

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

	var userName string
	if o.AllUsernames {
		if o.Username != "" {
			log.Logger().Warn("getting devpods for all usernames. Explicit username will be ignored")
		}
		// Leave userName blank
	} else {
		userName, err = o.GetUsername(o.Username)
		if err != nil {
			return err
		}
	}
	names, m, err := kube.GetDevPodNames(client, ns, userName)

	table := o.CreateTable()
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
