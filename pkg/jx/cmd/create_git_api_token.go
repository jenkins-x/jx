package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	create_git_token_long = templates.LongDesc(`
		Creates a new API Token for a user on a Git Server
`)

	create_git_token_example = templates.Examples(`
		# Add a new API Token for a user for the local Git server
        # prompting the user to find and enter the API Token
		jx create git token -n local someUserName

		# Add a new API Token for a user for the local Git server 
 		# using browser automation to login to the Git server
		# with the username and password to find the API Token
		jx create git token -n local -p somePassword someUserName	
	`)
)

// CreateGitTokenOptions the command line options for the command
type CreateGitTokenOptions struct {
	CreateOptions

	ServerFlags opts.ServerFlags
	Username    string
	Password    string
	ApiToken    string
	Timeout     string
}

// NewCmdCreateGitToken creates a command
func NewCmdCreateGitToken(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateGitTokenOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "token [username]",
		Short:   "Adds a new API token for a user on a Git server",
		Aliases: []string{"api-token"},
		Long:    create_git_token_long,
		Example: create_git_token_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.ServerFlags.AddGitServerFlags(cmd)
	cmd.Flags().StringVarP(&options.ApiToken, "api-token", "t", "", "The API Token for the user")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "The User password to try automatically create a new API Token")
	cmd.Flags().StringVarP(&options.Timeout, "timeout", "", "", "The timeout if using browser automation to generate the API token (by passing username and password)")

	return cmd
}

// Run implements the command
func (o *CreateGitTokenOptions) Run() error {
	args := o.Args
	if len(args) > 0 {
		o.Username = args[0]
	}
	if len(args) > 1 {
		o.ApiToken = args[1]
	}
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	server, err := o.FindGitServer(config, &o.ServerFlags)
	if err != nil {
		return err
	}
	err = o.EnsureGitServiceCRD(server)
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

	tokenUrl := gits.ProviderAccessTokenURL(server.Kind, server.URL, userAuth.Username)

	if userAuth.IsInvalid() && o.Password != "" {
		err = o.tryFindAPITokenFromBrowser(tokenUrl, userAuth)
	}

	if err != nil || userAuth.IsInvalid() {
		f := func(username string) error {
			tokenUrl := gits.ProviderAccessTokenURL(server.Kind, server.URL, username)

			log.Infof("Please generate an API Token for %s server %s\n", server.Kind, server.Label())
			log.Infof("Click this URL %s\n\n", util.ColorInfo(tokenUrl))
			log.Infof("Then COPY the token and enter in into the form below:\n\n")
			return nil
		}

		err = config.EditUserAuth(server.Label(), userAuth, o.Username, false, o.BatchMode, f, o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		if userAuth.IsInvalid() {
			return fmt.Errorf("You did not properly define the user authentication!")
		}
	}

	config.CurrentServer = server.URL
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}

	if config.PipeLineUsername == userAuth.Username {
		_, err = o.UpdatePipelineGitCredentialsSecret(server, userAuth)
		if err != nil {
			log.Warnf("Failed to update Jenkins X pipeline Git credentials secret: %v\n", err)
		}
	}

	log.Infof("Created user %s API Token for Git server %s at %s\n",
		util.ColorInfo(o.Username), util.ColorInfo(server.Name), util.ColorInfo(server.URL))

	return nil
}

// lets try use the users browser to find the API token
func (o *CreateGitTokenOptions) tryFindAPITokenFromBrowser(tokenUrl string, userAuth *auth.UserAuth) error {
	log.Info("No automation for launching browser presently")
	return fmt.Errorf("No automation to obtain API token via browser")
}


