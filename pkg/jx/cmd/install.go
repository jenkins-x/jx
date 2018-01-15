package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type InstallOptions struct {
	CommonOptions

	Domain             string
	GitProvider        string
	GitToken           string
	GitUser            string
	GitPass            string
	KubernetesProvider string
	CloudEnvRepository string
}

type Secrets struct {
	Login string
	Token string
}

const (
	JX_GIT_TOKEN    = "JX_GIT_TOKEN"
	JX_GIT_USER     = "JX_GIT_USER"
	JX_GIT_PASSWORD = "JX_GIT_PASSWORD"
)

var (
	instalLong = templates.LongDesc(`
		Installs the Jenkins X platform on a Kubernetes cluster

		Requires a --git-username and either --git-token or --git-password that can be used to create a new token.
		This is so the Jenkins-X platform can git tag your releases

`)

	instalExample = templates.Examples(`
		# Default installer which uses interactive prompts to generate git secrets
		jx install

		# Install with a GitHub personal access token
		jx install --git-username jenkins-x-bot --git-token 9fdbd2d070cd81eb12bca87861bcd850
`)
)

// NewCmdGet creates a command object for the generic "install" action, which
// installs the jenkins-x platform on a kubernetes cluster.
func NewCmdInstall(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {

	options := &InstallOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "install [flags]",
		Short:   "Install Jenkins-X",
		Long:    instalLong,
		Example: instalExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	cmd.Flags().StringVarP(&options.GitProvider, "git-provider", "", "github.com", "Git provider, used to create tokens if not provided.  Supported providers: [GitHub]")
	cmd.Flags().StringVarP(&options.GitToken, "git-token", "t", "", "Git token used to clone and tag releases, typically using a bot user account.  For GitHub use a personal access token with 'public_repo' scope, see https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line")
	cmd.Flags().StringVarP(&options.GitUser, "git-username", "u", "", "Git username used to tag releases in pipelines, typically this is a bot user")
	cmd.Flags().StringVarP(&options.GitPass, "git-password", "p", "", "Git username if a Personal Access Token should be created")
	cmd.Flags().StringVarP(&options.Domain, "domain", "d", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	cmd.Flags().StringVarP(&options.KubernetesProvider, "kubernetes-provider", "k", "minikube", "Service providing the kubernetes cluster.  Supported providers: [minikube,gke,thunder]")
	cmd.Flags().StringVarP(&options.CloudEnvRepository, "cloud-environment-repo", "c", "https://github.com/jenkins-x/cloud-environments", "Cloud Environments git repo")
	return cmd
}

// Run implements this command
func (options *InstallOptions) Run() error {
	// get secrets to use in helm install
	secrets, err := options.getGitSecrets()
	if err != nil {
		return err
	}

	// clone the environments repo
	wrkDir := filepath.Join(cmdutil.HomeDir(), ".jenkins-x", "cloud-environments")
	err = options.cloneJXCloudEnvironmentsRepo(wrkDir)
	if err != nil {
		return err
	}

	// run  helm install setting the token and domain values
	makefileDir := filepath.Join(wrkDir, fmt.Sprintf("env-%s", strings.ToLower(options.KubernetesProvider)))

	err = ioutil.WriteFile(filepath.Join(makefileDir, "secrets.yaml"), []byte(secrets), 0644)
	if err != nil {
		return err
	}

	makefile := exec.Command("make", "install")

	makefile.Dir = makefileDir
	makefile.Stdout = options.Out
	makefile.Stderr = options.Err
	err = makefile.Run()
	if err != nil {

		return err
	}

	log.Success("Jenkins-X installation completed successfully")
	return nil
}

// clones the jenkins-x cloud-environments repo to a local working dir
func (o *InstallOptions) cloneJXCloudEnvironmentsRepo(wrkDir string) error {
	log.Infof("Cloning the Jenkins-X cloud environments repo to %s\n", wrkDir)

	_, err := git.PlainClone(wrkDir, false, &git.CloneOptions{
		URL:           o.CloudEnvRepository,
		ReferenceName: "refs/heads/master",
		SingleBranch:  true,
		Progress:      o.Out,
	})
	if err != nil {
		if strings.Contains(err.Error(), "repository already exists") {
			log.Infof("A local Jenkins-X cloud environments repository already exists, recreate with latest? y/n: ")
			if log.AskForConfirmation(false) {
				err := os.RemoveAll(wrkDir)
				if err != nil {
					return err
				}

				return o.cloneJXCloudEnvironmentsRepo(wrkDir)
			}
		} else {
			return err
		}
	}
	return nil
}

// returns secrets that are used as values during the helm install
func (o *InstallOptions) getGitSecrets() (string, error) {
	username, token, err := o.getGitToken()
	if err != nil {
		return "", err
	}

	// TODO convert to a struct
	pipelineSecrets := `
PipelineSecrets:
  Netrc: |-
    machine github.com
      login %s
      password %s`
	return fmt.Sprintf(pipelineSecrets, username, token), nil
}

// returns the Git Token that should be used by Jenkins-X to setup credentials to clone repos and creates a secret for pipelines to tag a release
func (o *InstallOptions) getGitToken() (string, string, error) {
	username := o.GitUser
	if username == "" {
		if os.Getenv(JX_GIT_USER) != "" {
			username = os.Getenv(JX_GIT_USER)
		}
	}
	if username != "" {
		// first check git-token flag
		if o.GitToken != "" {
			return username, o.GitToken, nil
		}

		// second check for an environment variable
		if os.Getenv(JX_GIT_TOKEN) != "" {
			return username, os.Getenv(JX_GIT_TOKEN), nil
		}
	}

	o.Printf("Lets set up a git username and API token to be able to perform CI / CD\n\n")
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return "", "", err
	}
	config := authConfigSvc.Config()

	var server *auth.AuthServer
	gitProvider := o.GitProvider
	if gitProvider != "" {
		server = config.GetOrCreateServer(gitProvider)
	} else {
		server, err = config.PickServer("Which git provider?")
		if err != nil {
			return "", "", err
		}
	}
	url := server.URL
	userAuth, err := config.PickServerUserAuth(server, fmt.Sprintf("%s username for CI/CD pipelines:", server.Label()))
	if err != nil {
		return "", "", err
	}
	if userAuth.IsInvalid() {
		server.PrintGenerateAccessToken(o.Out)

		// TODO could we guess this based on the users ~/.git for github?
		defaultUserName := ""
		err = config.EditUserAuth(&userAuth, defaultUserName, false)
		if err != nil {
			return "", "", err
		}

		// TODO lets verify the auth works

		err = authConfigSvc.SaveUserAuth(url, &userAuth)
		if err != nil {
			return "", "", fmt.Errorf("Failed to store git auth configuration %s", err)
		}
		if userAuth.IsInvalid() {
			return "", "", fmt.Errorf("You did not properly define the user authentication!")
		}
	}
	return userAuth.Username, userAuth.ApiToken, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
