package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/users"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetUserOptions containers the CLI options
type GetUserOptions struct {
	GetOptions

	Pending bool
}

var (
	getUserLong = templates.LongDesc(`
		Display the Users
`)

	getUserExample = templates.Examples(`
		# List the users
		jx get user
	`)
)

// NewCmdGetUser creates the new command for: jx get env
func NewCmdGetUser(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetUserOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "users",
		Short:   "Display the User or Users the current user is a member of",
		Aliases: []string{"user"},
		Long:    getUserLong,
		Example: getUserExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Pending, "pending", "p", false, "Display only pending Users which are not yet provisioned yet")

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetUserOptions) Run() error {
	jxClient, ns, err := o.JXClientAndAdminNamespace()
	if err != nil {
		return err
	}
	users, names, err := users.GetUsers(jxClient, ns)
	if err != nil {
		return err
	}

	if len(names) == 0 {
		log.Info(`
There are no Users yet. Try create one via: jx create user
`)
		return nil
	}

	table := o.createTable()
	table.AddRow("LOGIN", "NAME", "EMAIL", "URL", "ROLES")
	for _, name := range names {
		user := users[name]
		if user != nil {
			spec := &user.Spec
			userKind := user.SubjectKind()
			userName := user.Name
			roleNames, err := kube.GetUserRoles(jxClient, ns, userKind, userName)
			if err != nil {
				log.Warnf("Failed to find User roles in namespace %s for User %s kind %s: %s\n", ns, userName, userKind, err)
			}
			table.AddRow(userName, spec.Name, spec.Email, spec.URL, strings.Join(roleNames, ", "))
		}
	}
	table.Render()
	return nil

}
