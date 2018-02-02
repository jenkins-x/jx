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
	delete_git_long = templates.LongDesc(`
		Deletes one or more git providers from your local settings
`)

	delete_git_example = templates.Examples(`
		# Deletes a git provider
		jx delete git MyProvider
	`)
)

// DeleteGitOptions the options for the create spring command
type DeleteGitOptions struct {
	CreateOptions

}

// NewCmdDeleteGit defines the command
func NewCmdDeleteGit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteGitOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "git",
		Short:   "Deletes one or more git providers",
		Aliases: []string{"git_provider"},
		Long:    delete_git_long,
		Example: delete_git_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	return cmd
}

// Run implements the command
func (o *DeleteGitOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return fmt.Errorf("Missing git provider name argument")
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	serverNames := config.GetServerNames()
	for _, arg := range args {
		idx := config.IndexOfServerName(arg)
		if idx <0  {
			return util.InvalidArg(arg, serverNames)
		}
		config.Servers = append(config.Servers[0:idx],  config.Servers[idx + 1:]...)
	}
	err = authConfigSvc.SaveConfig()
	if err != nil {
	  return err
	}
	o.Printf("Removed git providers: %s\n", util.ColorInfo(strings.Join(args, ", ")))
	return nil
}
