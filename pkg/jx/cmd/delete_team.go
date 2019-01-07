package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteTeamOptions are the flags for delete commands
type DeleteTeamOptions struct {
	CommonOptions

	SelectAll    bool
	SelectFilter string
	Confirm      bool
}

var (
	deleteTeamLong = templates.LongDesc(`
		Deletes one or more teams and their associated resources (Environments, Jenkins etc)
`)

	deleteTeamExample = templates.Examples(`
		# Delete the named team
		jx delete team cheese 

		# Delete the teams matching the given filter
		jx delete team -f foo 
	`)
)

// NewCmdDeleteTeam creates a command object
// retrieves one or more resources from a server.
func NewCmdDeleteTeam(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteTeamOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,

			Out: out,
			Err: errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "team",
		Short:   "Deletes one or more teams and their associated resources (Environments, Jenkins etc)",
		Long:    deleteTeamLong,
		Example: deleteTeamExample,
		Aliases: []string{"teams"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "a", false, "Should we default to selecting all the matched teams for deletion")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "f", "", "Filters the list of teams you can pick from")
	cmd.Flags().BoolVarP(&options.Confirm, "yes", "y", false, "Confirms we should uninstall this installation")
	return cmd
}

// Run implements this command
func (o *DeleteTeamOptions) Run() error {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}
	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	kube.RegisterEnvironmentCRD(apisClient)
	_, teamNames, err := kube.GetTeams(kubeClient)
	if err != nil {
		return err
	}

	names := o.Args
	if len(names) == 0 {
		if o.BatchMode {
			return fmt.Errorf("Missing team name argument")
		}
		names, err = util.SelectNamesWithFilter(teamNames, "Which teams do you want to delete: ", o.SelectAll, o.SelectFilter, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}

	if o.BatchMode {
		if !o.Confirm {
			return fmt.Errorf("In batch mode you must specify the '-y' flag to confirm")
		}
	} else {
		log.Warnf("You are about to delete the following teams '%s'. This operation CANNOT be undone!",
			strings.Join(names, ","))

		flag := false
		prompt := &survey.Confirm{
			Message: "Are you sure you want to delete all these teams?",
			Default: false,
		}
		err = survey.AskOne(prompt, &flag, nil, surveyOpts)
		if err != nil {
			return err
		}
		if !flag {
			return nil
		}
	}

	for _, name := range names {
		err = o.deleteTeam(name)
		if err != nil {
			log.Warnf("Failed to delete team %s: %s\n", name, err)
		}
	}
	return nil
}

func (o *DeleteTeamOptions) deleteTeam(name string) error {
	err := o.registerTeamCRD()
	if err != nil {
		return err
	}

	jxClient, ns, err := o.JXClientAndAdminNamespace()
	if err != nil {
		return err
	}
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}

	_, err = kubeClient.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if err != nil {
		// we don't have the namespace so the team cannot have been provisioned yet
		return kube.DeleteTeam(jxClient, ns, name)
	}
	origNamespace := o.currentNamespace
	o.ChangeNamespace(name)

	uninstall := &UninstallOptions{
		CommonOptions: o.CommonOptions,
		Namespace:     name,
		Force:         true,
	}
	uninstall.BatchMode = true

	err = o.ModifyTeam(ns, name, func(team *v1.Team) error {
		team.Status.ProvisionStatus = v1.TeamProvisionStatusDeleting
		team.Status.Message = "Deleting resources"
		return nil
	})
	if err != nil {
		return err
	}
	err = uninstall.Run()
	if err != nil {
		o.ModifyTeam(ns, name, func(team *v1.Team) error {
			team.Status.ProvisionStatus = v1.TeamProvisionStatusError
			team.Status.Message = fmt.Sprintf("Failed to delete team resources: %s", err)
			return nil
		})
	} else {
		err = kube.DeleteTeam(jxClient, ns, name)
	}
	o.ChangeNamespace(origNamespace)
	return err
}
