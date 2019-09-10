package get

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
)

// GetClusterOptions the command line options
type GetClusterOptions struct {
	GetOptions
	ClusterOptions opts.ClusterOptions
}

var (
	getClusterLong = templates.LongDesc(`
		Display the clusters in the current region/project

`)

	getClusterExample = templates.Examples(`
		# View the project configuration
		jx get clusters
	`)
)

// NewCmdGetCluster creates the command
func NewCmdGetCluster(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetClusterOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "clusters [flags]",
		Short:   "Display the current clusters",
		Long:    getClusterLong,
		Example: getClusterExample,
		Aliases: []string{"cluster"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	options.ClusterOptions.AddClusterFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetClusterOptions) Run() error {
	client, err := o.ClusterOptions.CreateClient()
	if err != nil {
		return err
	}
	clusters, err := client.List()

	table := o.CreateTable()
	table.AddRow("NAME", "LABELS", "STATUS")

	for _, cluster := range clusters {
		table.AddRow(cluster.Name, formatLabels(cluster.Labels), cluster.Status)
	}

	table.Render()
	return nil
}

func formatLabels(m map[string]string) string {
	builder := strings.Builder{}
	for k, v := range m {
		if builder.Len() > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(v)
	}
	return builder.String()
}
