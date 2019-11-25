package get

import (
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/util/maps"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
)

// GetClusterOptions the command line options
type GetClusterOptions struct {
	GetOptions
	ClusterOptions opts.ClusterOptions
	Filters        []string
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
	cmd.Flags().StringArrayVarP(&options.Filters, "filter", "f", nil, "The labels of the form 'key=value' to filter the clusters to choose from")
	return cmd
}

// Run implements this command
func (o *GetClusterOptions) Run() error {
	client, err := o.ClusterOptions.CreateClient(false)
	if err != nil {
		return err
	}
	filterLabels := maps.KeyValuesToMap(o.Filters)
	clusters, err := client.ListFilter(filterLabels)

	table := o.CreateTable()
	table.AddRow("NAME", "LOCATION", "LABELS", "STATUS")

	for _, cluster := range clusters {
		table.AddRow(cluster.Name, cluster.Location, maps.MapToString(cluster.Labels), cluster.Status)
	}

	table.Render()
	return nil
}
