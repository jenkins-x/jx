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
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Deletes one or more users 
`)

	deleteUserExample = templates.Examples(`
		# Delete the user with the login of cheese
		jx delete user cheese 
	`)
)

// NewCmdDeleteUser creates a command object
// retrieves one or more resources from a server.
func NewCmdDeleteUser(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteUserOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,

			Out: out,
			Err: errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "user",
		Short:   "Deletes one or more users",
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
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
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
		names, err = util.SelectNamesWithFilter(userNames, "Which users do you want to delete: ", o.SelectAll, o.SelectFilter, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}

	if o.BatchMode {
		if !o.Confirm {
			return fmt.Errorf("In batch mode you must specify the '-y' flag to confirm")
		}
	} else {
		log.Warnf("You are about to delete these users '%s' on the Git provider. This operation CANNOT be undone!",
			strings.Join(names, ","))

		flag := false
		prompt := &survey.Confirm{
			Message: "Are you sure you want to delete these all these users?",
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
		err = o.deleteUser(name)
		if err != nil {
			log.Warnf("Failed to delete user %s: %s\n", name, err)
		} else {
			log.Infof("Deleted user %s\n", util.ColorInfo(name))
		}
		log.Infof("Attempting to unbind user %s from associated role\n", util.ColorInfo(name))
		err = o.deleteUserFromRoleBinding(name, ns)
		if err != nil {
			log.Warnf("Problem to unbind user %s from associated role\n", util.ColorWarning(name))
		}
	}
	return nil
}

func (o *DeleteUserOptions) deleteUser(name string) error {
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}
	ns, err := kube.GetAdminNamespace(kubeClient, devNs)
	if err != nil {
		return err
	}
	return kube.DeleteUser(jxClient, ns, name)
}
func (o *DeleteUserOptions) deleteUserFromRoleBinding(name string, ns string) error {
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	foundUser := 0
	envRoleBindingsList, err := jxClient.JenkinsV1().EnvironmentRoleBindings(devNs).List(metav1.ListOptions{})
	for _, envRoleBinding := range envRoleBindingsList.Items {
		subjects := envRoleBinding.Spec.Subjects
		if subjects != nil {
			filteredEnvRoleBinding := subjects[:0]
			for _, subject := range subjects {
				if util.StringMatchesPattern(strings.Trim(name, ""), strings.Trim(subject.Name, "")) && util.StringMatchesPattern(strings.Trim(ns, ""), strings.Trim(subject.Namespace, "")) {
					subjectToDel := rbacv1.Subject{
						Name:      name,
						Namespace: ns,
					}
					filteredEnvRoleBinding = append(filteredEnvRoleBinding, subjectToDel)
					log.Infof("Found user %s to unbind from role\n", util.ColorInfo(name))
					foundUser = 1
					break
				}
			}
			if foundUser == 1 {
				break
			}
		}
	}
	return nil
}
