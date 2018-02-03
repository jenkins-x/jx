package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"strings"
)

var (
	delete_git_user_long = templates.LongDesc(`
		Deletes one or more git servers from your local settings
`)

	delete_git_user_example = templates.Examples(`
		# Deletes a git provider
		jx delete git server MyProvider
	`)
)

// DeleteGitUserOptions the options for the create spring command
type DeleteGitUserOptions struct {
	CreateOptions

	GitServerFlags GitServerFlags
}

// NewCmdDeleteGitUser defines the command
func NewCmdDeleteGitUser(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteGitUserOptions{
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
		Short:   "Deletes one or more git user api tokens",
		Aliases: []string{"token"},
		Long:    delete_git_user_long,
		Example: delete_git_user_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.GitServerFlags.addGitServerFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteGitUserOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return fmt.Errorf("Missing git user name")
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	server, err := o.findGitServer(config, &o.GitServerFlags)
	if err != nil {
		return err
	}
	for _, username := range args {
		err = server.DeleteUser(username)
		if err != nil {
			return err
		}
	}
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}
	o.Printf("Deleted API tokens for users: %s for git server %s at %s from local settings\n",
		util.ColorInfo(strings.Join(args, ", "), util.ColorInfo(server.Name), util.ColorInfo(server.URL)))
	return nil
}
