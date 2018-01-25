package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

var (
	create_git_long = templates.LongDesc(`
		Adds a new Git Repository Server URL
        ` + env_description + `
`)

	create_git_example = templates.Examples(`
		# Add a new git server URL
		jx create git https://my.server.com/
	`)
)

// CreateGitOptions the options for the create spring command
type CreateGitOptions struct {
	CreateOptions
}

// NewCmdCreateGit creates a command object for the "create" command
func NewCmdCreateGit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateGitOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "git [name] [url]",
		Short:   "Adds a new git hosting server URL",
		Aliases: []string{"gitserver"},
		Long:    create_git_long,
		Example: create_git_example,
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
func (o *CreateGitOptions) Run() error {
	args := o.Args
	if len(args) < 3 {
		return fmt.Errorf("Missing git server URL arguments. Usage: jx create git kind name url")
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
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}
	o.Printf("Added git server %s for URL %s\n", util.ColorInfo(name), util.ColorInfo(gitUrl))
	return nil
}
