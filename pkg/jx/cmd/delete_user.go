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

// DeleteUserOptions are the flags for delete commands
type DeleteUserOptions struct {
	CommonOptions

	SelectAll    bool
	SelectFilter string
	Confirm      bool
}

var (
	deleteUserLong = templates.LongDesc(`
		Deletes one or many users 
`)

	deleteUserExample = templates.Examples(`
		# Delete the user with the login of cheese
		jx delete user cheese 
	`)
)

// NewCmdDeleteUser creates a command object
// retrieves one or more resources from a server.
func NewCmdDeleteUser(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteUserOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "user",
		Short:   "Deletes one or many users",
		Long:    deleteUserLong,
		Example: deleteUserExample,
		Aliases: []string{"users"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "a", false, "Should we default to selecting all the matched users for deletion")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "f", "", "Fitlers the list of users you can pick from")
	cmd.Flags().BoolVarP(&options.Confirm, "yes", "y", false, "Confirms we should uninstall this installation")
	return cmd
}

// Run implements this command
func (o *DeleteUserOptions) Run() error {
	err := o.registerUserCRD()
	if err != nil {
		return err
	}

	jxClient, ns, err := o.JXClientAndAdminNamespace()
	if err != nil {
		return err
	}

	_, userNames, err := kube.GetUsers(jxClient, ns)
	if err != nil {
		return err
	}

	names := o.Args
	if len(names) == 0 {
		if o.BatchMode {
			return fmt.Errorf("Missing user login name argument")
		}
		names, err = util.SelectNamesWithFilter(userNames, "Which users do you want to delete: ", o.SelectAll, o.SelectFilter)
		if err != nil {
			return err
		}
	}

	if o.BatchMode {
		if !o.Confirm {
			return fmt.Errorf("In batch mode you must specify the '-y' flag to confirm")
		}
	} else {
		log.Warnf("You are about to delete these users '%s' on the git provider. This operation CANNOT be undone!",
			strings.Join(names, ","))

		flag := false
		prompt := &survey.Confirm{
			Message: "Are you sure you want to delete these all these users?",
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
		err = o.deleteUser(name)
		if err != nil {
			log.Warnf("Failed to delete user %s: %s\n", name, err)
		} else {
			log.Infof("Deleted user %s\n", util.ColorInfo(name))
		}
	}
	return nil
}

func (o *DeleteUserOptions) deleteUser(name string) error {
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	ns, err := kube.GetAdminNamespace(kubeClient, devNs)
	if err != nil {
		return err
	}
	return kube.DeleteUser(jxClient, ns, name)
}
func (o *DeleteUserOptions) deleteUserFromRoleBinding(name string, role string) error {
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	ns, err := kube.GetAdminNamespace(kubeClient, devNs)
	if err != nil {
		return err
	}
	return kube.DeleteUser(jxClient, ns, name)
}
