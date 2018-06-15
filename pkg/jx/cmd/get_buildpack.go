package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

// GetBuildPackOptions containers the CLI options
type GetBuildPackOptions struct {
	GetOptions
}

const (
	buildPack = "buildpack"
)

var (
	buildPacksAliases = []string{
		"build pack", "pack",
	}

	getBuildPackLong = templates.LongDesc(`
		Display the teams build pack git repository and references used for the current Team used on creating and importing projects

		For more documentation see: [https://jenkins-x.io/architecture/build-packs/](https://jenkins-x.io/architecture/build-packs/)
`)

	getBuildPackExample = templates.Examples(`
		# List the build pack  the current team
		jx get buildpack
	`)
)

// NewCmdGetBuildPack creates the new command for: jx get env
func NewCmdGetBuildPack(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetBuildPackOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     buildPack,
		Short:   "Display the teams build pack git repository and references used for the current Team used on creating and importing projects",
		Aliases: buildPacksAliases,
		Long:    getBuildPackLong,
		Example: getBuildPackExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetBuildPackOptions) Run() error {
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	table := o.CreateTable()
	table.AddRow("BUILD PACK GIT URL", "GIT REF")
	table.AddRow(settings.BuildPackURL, settings.BuildPackRef)
	table.Render()
	return nil
}
