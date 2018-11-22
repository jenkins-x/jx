package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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
		"build pack", "pack", "bp",
	}

	getBuildPackLong = templates.LongDesc(`
		Display the teams build pack Git repository and references used for the current Team used on creating and importing projects

		For more documentation see: [https://jenkins-x.io/architecture/build-packs/](https://jenkins-x.io/architecture/build-packs/)
`)

	getBuildPackExample = templates.Examples(`
		# List the build pack  the current team
		jx get buildpack
	`)
)

// NewCmdGetBuildPack creates the new command for: jx get env
func NewCmdGetBuildPack(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetBuildPackOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,

				Out: out,
				Err: errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     buildPack,
		Short:   "Display the teams build pack Git repository and references used for the current Team used on creating and importing projects",
		Aliases: buildPacksAliases,
		Long:    getBuildPackLong,
		Example: getBuildPackExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
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
