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
	delete_git_server_long = templates.LongDesc(`
		Deletes one or more git servers from your local settings
`)

	delete_git_server_example = templates.Examples(`
		# Deletes a git provider
		jx delete git server MyProvider
	`)
)

// DeleteGitServerOptions the options for the create spring command
type DeleteGitServerOptions struct {
	CommonOptions

	IgnoreMissingServer bool
}

// NewCmdDeleteGitServer defines the command
func NewCmdDeleteGitServer(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteGitServerOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "server",
		Short:   "Deletes one or more git server",
		Long:    delete_git_server_long,
		Example: delete_git_server_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.IgnoreMissingServer, "ignore-missing", "i", false, "Silently ignore attempts to remove a git server name that does not exist")
	return cmd
}

// Run implements the command
func (o *DeleteGitServerOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return fmt.Errorf("Missing git server name argument")
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	serverNames := config.GetServerNames()
	for _, arg := range args {
		idx := config.IndexOfServerName(arg)
		if idx < 0 {
			if o.IgnoreMissingServer {
				return nil
			}
			return util.InvalidArg(arg, serverNames)
		}
		config.Servers = append(config.Servers[0:idx], config.Servers[idx+1:]...)
	}
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}
	o.Printf("Deleted git servers: %s from local settings\n", util.ColorInfo(strings.Join(args, ", ")))
	return nil
}
