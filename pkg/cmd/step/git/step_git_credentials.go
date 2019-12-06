package git

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

const (
	optionOutputFile     = "output"
	optionGitHubAppOwner = "github-app-owner"
)

// StepGitCredentialsOptions contains the command line flags
type StepGitCredentialsOptions struct {
	step.StepOptions

	OutputFile     string
	GitHubAppOwner string
	GitKind        string
}

var (
	StepGitCredentialsLong = templates.LongDesc(`
		This pipeline step generates a Git credentials file for the current Git provider secrets

`)

	StepGitCredentialsExample = templates.Examples(`
		# generate the Git credentials file in the canonical location
		jx step git credentials

		# generate the Git credentials to a output file
		jx step git credentials -o /tmp/mycreds

`)
)

func NewCmdStepGitCredentials(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepGitCredentialsOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "credentials",
		Short:   "Creates the Git credentials file for the current pipeline",
		Long:    StepGitCredentialsLong,
		Example: StepGitCredentialsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.OutputFile, optionOutputFile, "o", "", "The output file name")
	cmd.Flags().StringVarP(&options.GitHubAppOwner, optionGitHubAppOwner, "g", "", "The owner (organisation or user name) if using GitHub App based tokens")
	cmd.Flags().StringVarP(&options.GitKind, "git-kind", "", "", "The git kind. e.g. github, bitbucketserver etc")
	return cmd
}

func (o *StepGitCredentialsOptions) Run() error {
	gha, err := o.IsGitHubAppMode()
	if err != nil {
		return err
	}
	if gha && o.GitHubAppOwner == "" {
		log.Logger().Infof("this command does nothing if using github app mode and no %s option specified", optionGitHubAppOwner)
		return nil
	}
	outFile := o.OutputFile
	if outFile == "" {
		// lets figure out the default output file
		cfgHome := os.Getenv("XDG_CONFIG_HOME")
		if cfgHome == "" {
			cfgHome = util.HomeDir()
		}
		if cfgHome != "" {
			outFile = filepath.Join(cfgHome, "git", "credentials")
		}
	}
	if outFile == "" {
		return util.MissingOption(optionOutputFile)
	}
	dir, _ := filepath.Split(outFile)
	if dir != "" {
		err := os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return err
		}
	}

	gitAuthSvc, err := o.GitAuthConfigServiceGitHubMode(gha, o.GitKind)
	if err != nil {
		return errors.Wrap(err, "creating git auth service")
	}
	return o.CreateGitCredentialsFile(outFile, gitAuthSvc)
}

// CreateGitCredentialsFile creates the git credentials into file using the provided auth config service
func (o *StepGitCredentialsOptions) CreateGitCredentialsFile(fileName string, configSvc auth.ConfigService) error {
	data, err := o.CreateGitCredentials(configSvc)
	if err != nil {
		return errors.Wrap(err, "creating git credentials")
	}
	if err := ioutil.WriteFile(fileName, data, util.DefaultWritePermissions); err != nil {
		return fmt.Errorf("Failed to write to %s: %s", fileName, err)
	}
	log.Logger().Infof("Generated Git credentials file %s", util.ColorInfo(fileName))
	return nil
}

// CreateGitCredentials creates the git credentials using the auth config service
func (o *StepGitCredentialsOptions) CreateGitCredentials(authConfigSvc auth.ConfigService) ([]byte, error) {
	cfg := authConfigSvc.Config()
	if cfg == nil {
		return nil, errors.New("no git auth config found")
	}

	var buffer bytes.Buffer
	for _, server := range cfg.Servers {
		auths := []*auth.UserAuth{}
		if o.GitHubAppOwner != "" {
			auths = server.Users
		} else {
			auth := server.CurrentAuth()
			if auth == nil {
				continue
			} else {
				auths = append(auths, auth)
			}
		}
		for _, auth := range auths {
			if o.GitHubAppOwner != "" && auth.GithubAppOwner != o.GitHubAppOwner {
				continue
			}
			username := auth.Username
			password := auth.ApiToken
			if password == "" {
				password = auth.BearerToken
			}
			if password == "" {
				password = auth.Password
			}
			if username == "" || password == "" {
				log.Logger().Warnf("Empty auth config for git service URL %q", server.URL)
				continue
			}
			u, err := url.Parse(server.URL)
			if err != nil {
				log.Logger().Warnf("Ignoring invalid git service URL %q", server.URL)
				continue
			}
			u.User = url.UserPassword(auth.Username, auth.ApiToken)
			buffer.WriteString(u.String() + "\n")
			// Write the https protocol in case only https is set for completeness
			if u.Scheme == "http" {
				u.Scheme = "https"
			}
			buffer.WriteString(u.String() + "\n")
		}
	}
	return buffer.Bytes(), nil
}
