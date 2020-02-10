package credentialhelper

import (
	"os"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/gits/credentialhelper"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
)

const (
	optionGitHubAppOwner = "github-app-owner"
)

// StepGitCredentialHelperOptions contains the command line flags
type StepGitCredentialHelperOptions struct {
	step.StepOptions

	GitKind        string
	GitHubAppOwner string
}

var (
	stepGitCredentialHelperLong = templates.LongDesc(`
		This pipeline step generates a Git credentials file for the current Git provider secrets

`)

	stepGitCredentialHelperExample = templates.Examples(`
		# respond to a git credentials request
		jx step git credentials-helper
`)
)

// NewCmdStepGitCredentialHelper creates a new command
func NewCmdStepGitCredentialHelper(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepGitCredentialHelperOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "credential-helper",
		Short:   "Creates a Git credentials helper",
		Long:    stepGitCredentialHelperLong,
		Example: stepGitCredentialHelperExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.GitKind, "git-kind", "", "", "The git kind. e.g. github, bitbucketserver etc")
	cmd.Flags().StringVarP(&options.GitHubAppOwner, optionGitHubAppOwner, "g", "", "The owner (organisation or user name) if using GitHub App based tokens")

	return cmd
}

// Run the main function
func (o *StepGitCredentialHelperOptions) Run() error {
	gha, err := o.IsGitHubAppMode()
	if err != nil {
		return err
	}

	if gha {
		log.Logger().Info("Running in GitHub App mode")
	} else {
		log.Logger().Info("Not running in GitHub App mode")
	}

	var authConfigSvc auth.ConfigService
	if gha {
		authConfigSvc, err = o.GitAuthConfigServiceGitHubAppMode(o.GitKind)
		if err != nil {
			return errors.Wrap(err, "when creating auth config service using GitAuthConfigServiceGitHubAppMode")
		}
	} else {
		authConfigSvc, err = o.GitAuthConfigService()
		if err != nil {
			return errors.Wrap(err, "when creating auth config service using GitAuthConfigService")
		}
	}

	credentials, err := o.CreateGitCredentialsFromAuthService(authConfigSvc)
	if err != nil {
		return errors.Wrap(err, "creating git credentials")
	}

	log.Logger().Infof("got credential %s", credentials)

	helper, err := credentialhelper.CreateGitCredentialsHelper(os.Stdin, os.Stdout, credentials)
	if err != nil {
		return errors.Wrap(err, "unable to create git credential helper")
	}

	// the credential helper operation (get|store|remove) is passed as last argument to the helper
	err = helper.Run(os.Args[len(os.Args)-1])
	if err != nil {
		return err
	}
	return nil
}

// CreateGitCredentialsFromAuthService creates the git credentials using the auth config service
func (o *StepGitCredentialHelperOptions) CreateGitCredentialsFromAuthService(authConfigSvc auth.ConfigService) ([]credentialhelper.GitCredential, error) {
	var credentialList []credentialhelper.GitCredential

	cfg := authConfigSvc.Config()
	if cfg == nil {
		return nil, errors.New("no git auth config found")
	}

	for _, server := range cfg.Servers {
		log.Logger().Infof("checking config for server %s at %s", server.Name, server.URL)
		log.Logger().Infof("found %d user auths", len(server.Users))

		for _, ua := range server.Users {
			log.Logger().Infof("Username: %s", ua.Username)
			log.Logger().Infof("GithubAppOwner: %s", ua.GithubAppOwner)
			log.Logger().Infof("ApiToken: %s", ua.ApiToken)
		}

		var auths []*auth.UserAuth
		if o.GitHubAppOwner != "" {
			auths = server.Users
		} else {
			gitAuth := server.CurrentAuth()
			if gitAuth == nil {
				continue
			} else {
				auths = append(auths, gitAuth)
			}
		}

		log.Logger().Infof("Got Auth: %d", len(auths))

		for _, gitAuth := range auths {
			log.Logger().Infof("Checking : '%s' with %s", o.GitHubAppOwner, gitAuth.GithubAppOwner)
			if o.GitHubAppOwner != "" && gitAuth.GithubAppOwner != o.GitHubAppOwner {
				continue
			}
			username := gitAuth.Username
			password := gitAuth.ApiToken
			if password == "" {
				password = gitAuth.BearerToken
			}
			if password == "" {
				password = gitAuth.Password
			}
			if username == "" || password == "" {
				log.Logger().Warnf("Empty auth config for git service URL %q", server.URL)
				continue
			}

			credential, err := credentialhelper.CreateGitCredentialFromURL(server.URL, username, password)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid git auth information")
			}

			credentialList = append(credentialList, credential)
		}
	}

	log.Logger().Infof("Got Cred List: %d", len(credentialList))
	return credentialList, nil
}
