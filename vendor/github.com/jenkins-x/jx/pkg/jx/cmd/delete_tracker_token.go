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
	deleteTrackerTokenLong = templates.LongDesc(`
		Deletes one or more API tokens for your issue tracker from your local settings
`)

	deleteTrackerTokenExample = templates.Examples(`
		# Deletes an issue tracker user token
		jx delete tracker token -n jira myusername
	`)
)

// DeleteTrackerTokenOptions the options for the create spring command
type DeleteTrackerTokenOptions struct {
	CreateOptions

	ServerFlags ServerFlags
}

// NewCmdDeleteTrackerToken defines the command
func NewCmdDeleteTrackerToken(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteTrackerTokenOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "token",
		Short:   "Deletes one or more api tokens for a user on an issue tracker server",
		Aliases: []string{"api-token"},
		Long:    deleteTrackerTokenLong,
		Example: deleteTrackerTokenExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.ServerFlags.addGitServerFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteTrackerTokenOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return fmt.Errorf("Missing issue tracker user name")
	}
	authConfigSvc, err := o.CreateIssueTrackerAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	server, err := o.findIssueTrackerServer(config, &o.ServerFlags)
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
	o.Printf("Deleted API tokens for users: %s for issue tracker server %s at %s from local settings\n",
		util.ColorInfo(strings.Join(args, ", ")), util.ColorInfo(server.Name), util.ColorInfo(server.URL))
	return nil
}
