package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	create_git_server_long = templates.LongDesc(`
		Adds a new Git Server URL
        ` + env_description + `
`)

	create_git_server_example = templates.Examples(`
		# Add a new git server URL
		jx create git server gitea mythingy https://my.server.com/
	`)
)

// CreateGitServerOptions the options for the create spring command
type CreateGitServerOptions struct {
	CreateOptions
}

// NewCmdCreateGitServer creates a command object for the "create" command
func NewCmdCreateGitServer(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateGitServerOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "server [kind] [name] [url]",
		Short:   "Adds a new git server URL",
		Aliases: []string{"gitserver"},
		Long:    create_git_server_long,
		Example: create_git_server_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	//addCreateAppFlags(cmd, &options.CreateOptions)

	return cmd
}

// Run implements the command
func (o *CreateGitServerOptions) Run() error {
	args := o.Args
	if len(args) < 3 {
		return fmt.Errorf("Missing git server URL arguments. Usage: jx create git server kind name url")
	}
	kind := args[0]
	name := args[1]
	gitUrl := args[2]
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()
	config.GetOrCreateServerName(gitUrl, name, kind)
	config.CurrentServer = gitUrl
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}
	o.Printf("Added git server %s for URL %s\n", util.ColorInfo(name), util.ColorInfo(gitUrl))
	return nil
}
