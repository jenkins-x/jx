package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetBuildPackOptions containers the CLI options
type GetBuildPackOptions struct {
	GetOptions

	All bool
}

const (
	buildPack = "buildpack"
)

var (
	buildPacksAliases = []string{
		"build pack", "pack", "bp",
	}

	getBuildPackLong = templates.LongDesc(`
		Display the teams build pack Git repository and references used when creating and importing projects

		For more documentation see: [https://jenkins-x.io/architecture/build-packs/](https://jenkins-x.io/architecture/build-packs/)
`)

	getBuildPackExample = templates.Examples(`
		# List the build pack for the current team
		jx get buildpack

		# List all the available build packs you can pick from
		jx get bp -a
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

	cmd.Flags().BoolVarP(&options.All, "all", "a", false, "View all available Build Packs")

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetBuildPackOptions) Run() error {
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	table := o.createTable()
	if o.All {
		jxClient, ns, err := o.JXClientAndDevNamespace()
		if err != nil {
			return err
		}
		m, names, err := builds.GetBuildPacks(jxClient, ns)
		if err != nil {
			return err
		}
		table.AddRow("BUILD PACK", "GIT URL", "GIT REF", "DEFAULT")
		for _, name := range names {
			bp := m[name]
			if bp != nil {
				label := bp.Spec.Label
				gitURL := bp.Spec.GitURL
				gitRef := bp.Spec.GitRef
				defaultPack := ""
				if gitURL == settings.BuildPackURL {
					defaultPack = "  " + util.CheckMark()
				}
				table.AddRow(label, gitURL, gitRef, defaultPack)
			}
		}
	} else {
		table.AddRow(settings.BuildPackName, settings.BuildPackURL, settings.BuildPackRef)
	}
	table.Render()
	return nil
}
