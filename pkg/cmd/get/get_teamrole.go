package get

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/spf13/cobra"
)

// GetTeamRoleOptions containers the CLI options
type GetTeamRoleOptions struct {
	Options
}

var (
	getTeamRoleLong = templates.LongDesc(`
		Display the roles for members of a Team
`)

	getTeamRoleExample = templates.Examples(`
		# List the team roles for the current team
		jx get teamrole

	`)
)

// NewCmdGetTeamRole creates the new command for: jx get env
func NewCmdGetTeamRole(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetTeamRoleOptions{
		Options: Options{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "teamroles",
		Short:   "Display the Team or Teams the current user is a member of",
		Aliases: []string{"teamrole"},
		Long:    getTeamRoleLong,
		Example: getTeamRoleExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	options.AddGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetTeamRoleOptions) Run() error {
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	teamRoles, names, err := kube.GetTeamRoles(kubeClient, ns)
	if err != nil {
		return err
	}
	if len(teamRoles) == 0 {
		log.Logger().Info(`
There are no Team roles defined so far!
`)
		return nil
	}

	table := o.CreateTable()
	table.AddRow("NAME", "TITLE", "DESCRIPTION")
	for _, name := range names {
		title := ""
		description := ""
		teamRole := teamRoles[name]
		if teamRole != nil {
			ann := teamRole.Annotations
			if ann != nil {
				title = ann[kube.AnnotationTitle]
				description = ann[kube.AnnotationDescription]
			}
		}
		table.AddRow(name, title, description)
	}
	table.Render()
	return nil
}
