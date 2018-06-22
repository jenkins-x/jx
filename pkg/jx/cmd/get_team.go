package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
)

// GetTeamOptions containers the CLI options
type GetTeamOptions struct {
	GetOptions
}

var (
	getTeamLong = templates.LongDesc(`
		Display the Team or Teams a user is a member of.
`)

	getTeamExample = templates.Examples(`
		# List the team or teams the current user is a member of
		jx get team
	`)
)

// NewCmdGetTeam creates the new command for: jx get env
func NewCmdGetTeam(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetTeamOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "teams",
		Short:   "Display the Team or Teams the current user is a member of",
		Aliases: []string{"team"},
		Long:    getTeamLong,
		Example: getTeamExample,
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
func (o *GetTeamOptions) Run() error {
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	teams, _, err := kube.GetTeams(kubeClient)
	if err != nil {
		return err
	}
	if len(teams) == 0 {
		log.Info(`
You do not belong to any teams.
Have you installed Jenkins X yet to create a team?
See https://jenkins-x.io/getting-started/\n for more detail
`)
		return nil
	}

	table := o.CreateTable()
	table.AddRow("NAME")
	for _, team := range teams {
		table.AddRow(team.Name)
	}
	table.Render()
	return nil
}
