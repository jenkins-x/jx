package cmd

import (
	"fmt"

	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/sirupsen/logrus"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
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

	ServerFlags opts.ServerFlags
}

// NewCmdDeleteTrackerToken defines the command
func NewCmdDeleteTrackerToken(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &DeleteTrackerTokenOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "token",
		Short:   "Deletes one or more API tokens for a user on an issue tracker server",
		Aliases: []string{"api-token"},
		Long:    deleteTrackerTokenLong,
		Example: deleteTrackerTokenExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.ServerFlags.AddGitServerFlags(cmd)
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

	server, err := o.FindIssueTrackerServer(config, &o.ServerFlags)
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
	logrus.Infof("Deleted API tokens for users: %s for issue tracker server %s at %s from local settings\n",
		util.ColorInfo(strings.Join(args, ", ")), util.ColorInfo(server.Name), util.ColorInfo(server.URL))
	return nil
}
