package cmd

import (
	"io"

	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	create_git_user_long = templates.LongDesc(`
		Adds a new Git Server URL
        ` + env_description + `
`)

	create_git_user_example = templates.Examples(`
		# Add a new git server URL
		jx create git server gitea mythingy https://my.server.com/
	`)
)

// CreateGitUserOptions the command line options for the command
type CreateGitUserOptions struct {
	CreateOptions

	GitServerFlags GitServerFlags
	Username       string
	Password       string
	ApiToken       string
}

// NewCmdCreateGitUser creates a command
func NewCmdCreateGitUser(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateGitUserOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "user [username]",
		Short:   "Adds a new user name and api token for a git server server",
		Aliases: []string{"token"},
		Long:    create_git_user_long,
		Example: create_git_user_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	options.GitServerFlags.addGitServerFlags(cmd)
	cmd.Flags().StringVarP(&options.ApiToken, "--api-token", "t", "", "The API Token for the user")

	return cmd
}

// Run implements the command
func (o *CreateGitUserOptions) Run() error {
	args := o.Args
	if len(args) > 0 {
		o.Username = args[0]
	}
	if len(args) > 1 {
		o.ApiToken = args[1]
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

	// TODO add the API thingy...
	if o.Username == "" {
		return fmt.Errorf("No Username specified")
	}

	userAuth := config.GetOrCreateUserAuth(server.URL, o.Username)
	if o.ApiToken != "" {
		userAuth.ApiToken = o.ApiToken
	}

	if userAuth.IsInvalid() {
		tokenUrl := gits.ProviderAccessTokenURL(server.Kind, server.URL)

		o.Printf("Please generate an API Token for server %s\n", server.Label())
		o.Printf("Click this URL %s\n\n", util.ColorInfo(tokenUrl))
		o.Printf("Then COPY the token and enter in into the form below:\n\n")

		err = config.EditUserAuth(userAuth, o.Username, false)
		if err != nil {
			return err
		}
		if userAuth.IsInvalid() {
			return fmt.Errorf("You did not properly define the user authentication!")
		}
	}

	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}
	o.Printf("Added user %s API Token for git server %s at %s\n",
		util.ColorInfo(o.Username), util.ColorInfo(server.Name), util.ColorInfo(server.URL))
	return nil
}
