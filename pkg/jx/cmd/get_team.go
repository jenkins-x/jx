package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetTeamOptions containers the CLI options
type GetTeamOptions struct {
	GetOptions

	Pending bool
}

var (
	getTeamLong = templates.LongDesc(`
		Display the Team or Teams a user is a member of.
`)

	getTeamExample = templates.Examples(`
		# List the provisioned team or teams the current user is a member of
		jx get team

		# List the pending Teams which are not yet provisioned and available for use
		jx get team -p
	`)
)

// NewCmdGetTeam creates the new command for: jx get env
func NewCmdGetTeam(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetTeamOptions{
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
		Use:     "teams",
		Short:   "Display the Team or Teams the current user is a member of",
		Aliases: []string{"team"},
		Long:    getTeamLong,
		Example: getTeamExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Pending, "pending", "p", false, "Display only pending Teams which are not yet provisioned yet")

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetTeamOptions) Run() error {
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}
	if o.Pending {
		return o.getPendingTeams()
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

	table := o.createTable()
	table.AddRow("NAME")
	for _, team := range teams {
		table.AddRow(team.Name)
	}
	table.Render()
	return nil
}

func (o *GetTeamOptions) getPendingTeams() error {
	err := o.registerTeamCRD()
	if err != nil {
		return err
	}

	jxClient, ns, err := o.JXClientAndAdminNamespace()
	if err != nil {
		return err
	}

	teams, names, err := kube.GetPendingTeams(jxClient, ns)
	if err != nil {
		return err
	}

	if len(names) == 0 {
		log.Info(`
There are no pending Teams yet. Try create one via: jx create team --pending
`)
		return nil
	}

	table := o.createTable()
	table.AddRow("NAME", "STATUS", "KIND", "MEMBERS")
	for _, team := range teams {
		spec := &team.Spec
		table.AddRow(team.Name, string(team.Status.ProvisionStatus), string(spec.Kind), strings.Join(spec.Members, ", "))
	}
	table.Render()
	return nil

}
