package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"strings"
)

// GetAddonOptions the command line options
type GetAddonOptions struct {
	GetOptions
}

var (
	get_addon_long = templates.LongDesc(`
		Display the available addons

`)

	get_addon_example = templates.Examples(`
		# List all the possible addons
		jx get addon
	`)
)

// NewCmdGetAddon creates the command
func NewCmdGetAddon(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetAddonOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "addon [flags]",
		Short:   "Lists the addons",
		Long:    get_addon_long,
		Example: get_addon_example,
		Aliases: []string{"addons", "add-on"},
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
func (o *GetAddonOptions) Run() error {
	statusMap := map[string]string{}

	output, err := o.getCommandOutput("", "helm", "list")
	if err != nil {
		o.warnf("Failed to find helm installs: %s\n", err)
	} else {
		for _, line := range strings.Split(output, "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) > 3 {
				statusMap[strings.TrimSpace(fields[0])] = fields[3]
			}
		}
	}

	charts := kube.AddonCharts

	table := o.CreateTable()
	table.AddRow("NAME", "CHART", "STATUS")

	keys := util.SortedMapKeys(charts)
	for _, k := range keys {
		chart := charts[k]
		status := statusMap[k]
		table.AddRow(k, chart, status)
	}

	table.Render()
	return nil
}
