package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"io"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	optionLogin = "login"
)

var (
	createUserLong = templates.LongDesc(`
		Creates a user
`)

	createUserExample = templates.Examples(`
		# Create an issue in the current project
		jx create issue -t "something we should do"


		# Create an issue with a title and a body
		jx create issue -t "something we should do" --body "	
		some more
		text
		goes
		here
		""
"
	`)
)

// CreateUserOptions the options for the create spring command
type CreateUserOptions struct {
	CreateOptions
	Role     string
	UserSpec v1.UserDetails
}

// NewCmdCreateUser creates a command object for the "create" command
func NewCmdCreateUser(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateUserOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "user",
		Short:   "Create a new User which is then provisioned by the user controller",
		Aliases: []string{"env"},
		Long:    createUserLong,
		Example: createUserExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.UserSpec.Login, optionLogin, "l", "", "The user login name")
	cmd.Flags().StringVarP(&options.UserSpec.Name, "name", "n", "", "The textual full name of the user")
	cmd.Flags().StringVarP(&options.UserSpec.Email, "email", "e", "", "The users email address")
	cmd.Flags().StringVarP(&options.Role, "role", "r", "", "The user's role")

	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreateUserOptions) Run() error {
	err := o.registerUserCRD()
	if err != nil {
		return err
	}

	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	ns, err := kube.GetAdminNamespace(kubeClient, devNs)
	if err != nil {
		return err
	}

	_, names, err := kube.GetUsers(jxClient, ns)
	if err != nil {
		return err
	}

	spec := &o.UserSpec
	login := spec.Login
	if login == "" {
		args := o.Args
		if len(args) > 0 {
			login = args[0]
		}
	}
	if login == "" {
		return util.MissingOption(optionLogin)
	}

	if util.StringArrayIndex(names, login) >= 0 {
		return fmt.Errorf("The User %s already exists!", login)
	}

	name := spec.Name
	if name == "" {
		name = strings.Title(login)
	}
	user := kube.CreateUser(ns, login, name, spec.Email)
	_, err = jxClient.JenkinsV1().Users(ns).Create(user)
	if err != nil {
		return fmt.Errorf("Failed to create User %s: %s", login, err)
	}
	log.Infof("Created User: %s\n", util.ColorInfo(login))
	log.Infof("Binding user %s with role: %s\n", util.ColorInfo(login), o.Role)

	envRoleBindingsList, err := jxClient.JenkinsV1().EnvironmentRoleBindings(ns).List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Failed to retrieve environment role binding list for namespace %s: %s", ns, err)
	}

	for _, envRoleBinding := range envRoleBindingsList.Items {
		if util.StringMatchesPattern(o.Role, envRoleBinding.Spec.RoleRef.Name) {
			log.Infof("Role %s exists, binding user %s with role.\n", util.ColorInfo(o.Role), util.ColorInfo(login))
			newSubject := rbacv1.Subject{
				Name:      o.UserSpec.Name,
				Kind:      "User", //TODO: should the default be user? Should we also pass kind as part of user creation step?
				Namespace: ns,
			}
			envRoleBinding.Spec.Subjects = append(envRoleBinding.Spec.Subjects, newSubject)
		} else {
			log.Warnf("Role %s doesn't exist, will not bind user %s with role\n", util.ColorWarning(o.Role), util.ColorWarning(login))
		}
		return nil
	}
}
