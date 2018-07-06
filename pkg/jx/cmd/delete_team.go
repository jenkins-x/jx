package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
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
		Deletes one or many teams and their associated resources (Environments, Jenkins etc)
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
func NewCmdDeleteTeam(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteTeamOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "team",
		Short:   "Deletes one or many teams and their associated resources (Environments, Jenkins etc)",
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
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "f", "", "Fitlers the list of teams you can pick from")
	cmd.Flags().BoolVarP(&options.Confirm, "yes", "y", false, "Confirms we should uninstall this installation")
	return cmd
}

// Run implements this command
func (o *DeleteTeamOptions) Run() error {
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	apisClient, err := o.CreateApiExtensionsClient()
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
		names, err = util.SelectNamesWithFilter(teamNames, "Which teams do you want to delete: ", o.SelectAll, o.SelectFilter)
		if err != nil {
			return err
		}
	}

	if o.BatchMode {
		if !o.Confirm {
			return fmt.Errorf("In batch mode you must specify the '-y' flag to confirm")
		}
	} else {
		log.Warnf("You are about to delete these teams '%s' on the git provider. This operation CANNOT be undone!",
			strings.Join(names, ","))

		flag := false
		prompt := &survey.Confirm{
			Message: "Are you sure you want to delete these all these teams?",
			Default: false,
		}
		err = survey.AskOne(prompt, &flag, nil)
		if err != nil {
			return err
		}
		if !flag {
			return nil
		}
	}

	for _, name := range names {
		uninstall := &UninstallOptions{
			CommonOptions: o.CommonOptions,
			Namespace:     name,
			Confirm:       true,
		}
		uninstall.BatchMode = true

		o.changeNamespace(name)
		err = uninstall.Run()
		if err != nil {
			log.Warnf("Failed to delete team %s\n", name)
		}
		o.changeNamespace("default")
	}
	return nil
}

func (o *DeleteTeamOptions) changeNamespace(ns string) {
	nsOptions := &NamespaceOptions{
		CommonOptions: o.CommonOptions,
	}
	nsOptions.BatchMode = true
	nsOptions.Args = []string{ns}
	err := nsOptions.Run()
	if err != nil {
		log.Warnf("Failed to set context to namespace %s: %s", ns, err)
	}
}
