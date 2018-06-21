package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

var (
	editBuildpackLong = templates.LongDesc(`
		Edits the build pack configuration for your team
`)

	editBuildpackExample = templates.Examples(`
		# Edit the build pack configuration for your team
		jx edit buildpack

		For more documentation see: [https://jenkins-x.io/architecture/build-packs/](https://jenkins-x.io/architecture/build-packs/)
	`)
)

// EditBuildpackOptions the options for the create spring command
type EditBuildpackOptions struct {
	EditOptions

	BuildPackURL string
	BuildPackRef string
}

// NewCmdEditBuildpack creates a command object for the "create" command
func NewCmdEditBuildpack(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &EditBuildpackOptions{
		EditOptions: EditOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     buildPack,
		Short:   "Edits the build pack configuration for your team",
		Aliases: buildPacksAliases,
		Long:    editBuildpackLong,
		Example: editBuildpackExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.BuildPackURL, "url", "u", "", "The URL for the build pack git repository")
	cmd.Flags().StringVarP(&options.BuildPackRef, "ref", "r", "", "The git reference (branch,tag,sha) in the git repository touse")
	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *EditBuildpackOptions) Run() error {
	buildPackURL := o.BuildPackURL
	BuildPackRef := o.BuildPackRef

	if !o.BatchMode {
		teamSettings, err := o.TeamSettings()
		if err != nil {
			return err
		}
		if buildPackURL == "" {
			buildPackURL, err = util.PickValue("Build pack git clone URL:", teamSettings.BuildPackURL, true)
			if err != nil {
				return err
			}
		}
		if BuildPackRef == "" {
			BuildPackRef, err = util.PickValue("Build pack git ref:", teamSettings.BuildPackRef, true)
			if err != nil {
				return err
			}
		}
	}

	callback := func(env *v1.Environment) error {
		teamSettings := &env.Spec.TeamSettings
		if buildPackURL != "" {
			teamSettings.BuildPackURL = buildPackURL
		}
		if BuildPackRef != "" {
			teamSettings.BuildPackRef = BuildPackRef
		}
		log.Infof("Setting the team build pack to repo: %s ref: %s\n", util.ColorInfo(buildPackURL), util.ColorInfo(BuildPackRef))
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
