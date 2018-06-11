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
`)

	create_git_server_example = templates.Examples(`
		# Add a new git server URL
		jx create git server gitea
	`)

	gitKindToServiceName = map[string]string{
		"gitea": "gitea-gitea",
	}
)

// CreateGitServerOptions the options for the create spring command
type CreateGitServerOptions struct {
	CreateOptions

	Name string
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
		Use:     "server kind [url]",
		Short:   "Creates a new git server URL",
		Aliases: []string{"provider"},
		Long:    create_git_server_long,
		Example: create_git_server_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "The name for the git server being created")
	return cmd
}

// Run implements the command
func (o *CreateGitServerOptions) Run() error {
	args := o.Args
	if len(args) < 1 {
		return missingGitServerArguments()
	}
	kind := args[0]
	name := o.Name
	if name == "" {
		name = kind
	}
	gitUrl := ""
	if len(args) > 1 {
		gitUrl = args[1]
	} else {
		// lets try find the git URL based on the provider
		serviceName := gitKindToServiceName[kind]
		if serviceName != "" {
			url, err := o.findService(serviceName)
			if err != nil {
				return fmt.Errorf("Failed to find %s git service %s: %s", kind, serviceName, err)
			}
			gitUrl = url
		}
	}

	if gitUrl == "" {
		return missingGitServerArguments()
	}
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

func missingGitServerArguments() error {
	return fmt.Errorf("Missing git server URL arguments. Usage: jx create git server kind [url]")
}
