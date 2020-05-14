package version

import (
	"github.com/jenkins-x/jx/v2/pkg/version"

	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/spf13/cobra"
)

// Options struct contains the version flags
type Options struct {
	*opts.CommonOptions
	short bool
}

// NewCmdVersion creates the version command
func NewCmdVersion(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &Options{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().BoolVar(&options.short, "short", false, "Print version")
	return cmd
}

//Run implements the version command
func (o *Options) Run() error {
	info := util.ColorInfo
	table := o.CreateTable()
	jxVersion := version.GetVersion()
	switch o.short {
	case true:
		table.AddRow("Version", info(jxVersion))
	default:
		commit := version.GetRevision()
		treeState := version.GetTreeState()
		buildDate := version.GetBuildDate()
		goVersion := version.GetGoVersion()
		table.AddRow("Version", info(jxVersion))
		table.AddRow("Commit", info(commit))
		table.AddRow("Build date", info(buildDate))
		table.AddRow("Go version", info(goVersion))
		table.AddRow("Git tree state", info(treeState))
	}
	table.Render()
	return nil
}
