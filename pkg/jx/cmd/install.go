package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/src-d/go-git.v4"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type InstallOptions struct {
	CommonOptions

	Domain             string
	HTTPS              bool
	GitProvider        string
	GitToken           string
	GitUser            string
	GitPass            string
	Provider           string
	CloudEnvRepository string
	LocalHelmRepoName  string
}

type Secrets struct {
	Login string
	Token string
}

const (
	JX_GIT_TOKEN                   = "JX_GIT_TOKEN"
	JX_GIT_USER                    = "JX_GIT_USER"
	DEFAULT_CLOUD_ENVIRONMENTS_URL = "https://github.com/jenkins-x/cloud-environments"

	GitSecretsFile  = "gitSecrets.yaml"
	ExtraValuesFile = "extraValues.yaml"
)

var (
	instalLong = templates.LongDesc(`
		Installs the Jenkins X platform on a Kubernetes cluster

		Requires a --git-username and either --git-token or --git-password that can be used to create a new token.
		This is so the Jenkins X platform can git tag your releases

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
		Short:   "Install Jenkins X",
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
	cmd.Flags().BoolVarP(&options.HTTPS, "https", "", false, "Instructs Jenkins X to generate https not http Ingress rules")
	cmd.Flags().StringVarP(&options.Provider, "provider", "", "", "Cloud service providing the kubernetes cluster.  Supported providers: [minikube,gke,aks]")
	cmd.Flags().StringVarP(&options.CloudEnvRepository, "cloud-environment-repo", "c", DEFAULT_CLOUD_ENVIRONMENTS_URL, "Cloud Environments git repo")
	cmd.Flags().StringVarP(&options.LocalHelmRepoName, "local-helm-repo-name", "", kube.LocalHelmRepoName, "The name of the helm repository for the installed Chart Museum")

	return cmd
}

// Run implements this command
func (options *InstallOptions) Run() error {
	client, _, err := options.Factory.CreateClient()
	if err != nil {
		return err
	}
	options.kubeClient = client

	options.Provider, err = options.GetCloudProvider(options.Provider)
	if err != nil {
		return err
	}

	// get secrets to use in helm install
	secrets, err := options.getGitSecrets()
	if err != nil {
		return err
	}

	config, err := options.getExposecontrollerConfigValues()
	if err != nil {
		return err
	}

	// clone the environments repo
	wrkDir := filepath.Join(util.HomeDir(), ".jx", "cloud-environments")
	err = options.cloneJXCloudEnvironmentsRepo(wrkDir)
	if err != nil {
		return err
	}

	// run  helm install setting the token and domain values
	if options.Provider == "" {
		return fmt.Errorf("No kubernetes provider found to match cloud-environment with")
	}
	makefileDir := filepath.Join(wrkDir, fmt.Sprintf("env-%s", strings.ToLower(options.Provider)))
	if _, err := os.Stat(wrkDir); os.IsNotExist(err) {
		return fmt.Errorf("cloud environment dir %s not found", makefileDir)
	}

	// create a temporary file that's used to pass current git creds to helm in order to create a secret for pipelines to tag releases
	dir, err := util.ConfigDir()
	if err != nil {
		return err
	}

	secretsFileName := filepath.Join(dir, GitSecretsFile)
	err = ioutil.WriteFile(secretsFileName, []byte(secrets), 0644)
	if err != nil {
		return err
	}

	configFileName := filepath.Join(dir, ExtraValuesFile)
	err = ioutil.WriteFile(configFileName, []byte(config), 0644)
	if err != nil {
		return err
	}
	arg := fmt.Sprintf("ARGS=--values=%s --values=%s", secretsFileName, configFileName)

	// run the helm install
	err = options.runCommandFromDir(makefileDir, "make", arg, "install")
	if err != nil {
		return err
	}

	// cleanup temporary files
	err = os.Remove(secretsFileName)
	if err != nil {
		return err
	}

	err = os.Remove(configFileName)
	if err != nil {
		return err
	}

	err = options.waitForInstallToBeReady()
	if err != nil {
		return err
	}

	err = options.registerLocalHelmRepo(options.LocalHelmRepoName)
	if err != nil {
		return err
	}

	err = options.runCommand("kubectl", "get", "ingress")
	if err != nil {
		return err
	}

	log.Success("\nJenkins X installation completed successfully")

	options.Printf("\nTo import existing projects into Jenkins: %s\n", util.ColorInfo("jx import"))
	options.Printf("To create a new Spring Boot microservice: %s\n", util.ColorInfo("jx create spring -d web -d actuator"))
	return nil
}

// clones the jenkins-x cloud-environments repo to a local working dir
func (o *InstallOptions) cloneJXCloudEnvironmentsRepo(wrkDir string) error {
	log.Infof("Cloning the Jenkins X cloud environments repo to %s\n", wrkDir)
	if o.CloudEnvRepository == "" {
		return fmt.Errorf("No cloud environment git URL")
	}
	_, err := git.PlainClone(wrkDir, false, &git.CloneOptions{
		URL:           o.CloudEnvRepository,
		ReferenceName: "refs/heads/master",
		SingleBranch:  true,
		Progress:      o.Out,
	})
	if err != nil {
		if strings.Contains(err.Error(), "repository already exists") {
			confirm := &survey.Confirm{
				Message: "A local Jenkins X cloud environments repository already exists, recreate with latest?",
				Default: true,
			}
			flag := false
			err := survey.AskOne(confirm, &flag, nil)
			if err != nil {
				return err
			}
			if flag {
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

	server := o.GitProvider
	if server == "" {
		return "", fmt.Errorf("No Git Server found")
	}

	// TODO convert to a struct
	pipelineSecrets := `
PipelineSecrets:
  GitCreds: |-
    https://%s:%s@%s`
	return fmt.Sprintf(pipelineSecrets, username, token, server), nil
}

func (o *InstallOptions) getExposecontrollerConfigValues() (string, error) {
	var err error
	o.Domain, err = o.GetDomain(o.kubeClient, o.Domain, o.Provider)
	if err != nil {
		return "", err
	}
	// TODO convert to a struct
	config := `
expose:
  Args:
    - --exposer
    - Ingress
    - --http
    - "%v"
    - --domain
    - %s
`
	return fmt.Sprintf(config, !o.HTTPS, o.Domain), nil
}

// returns the Git Token that should be used by Jenkins X to setup credentials to clone repos and creates a secret for pipelines to tag a release
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
		gits.PrintCreateRepositoryGenerateAccessToken(server, o.Out)

		// TODO could we guess this based on the users ~/.git for github?
		defaultUserName := ""
		err = config.EditUserAuth(userAuth, defaultUserName, false)
		if err != nil {
			return "", "", err
		}

		// TODO lets verify the auth works

		err = authConfigSvc.SaveUserAuth(url, userAuth)
		if err != nil {
			return "", "", fmt.Errorf("Failed to store git auth configuration %s", err)
		}
		if userAuth.IsInvalid() {
			return "", "", fmt.Errorf("You did not properly define the user authentication!")
		}
	}
	return userAuth.Username, userAuth.ApiToken, nil
}

func (o *InstallOptions) waitForInstallToBeReady() error {
	f := o.Factory
	client, ns, err := f.CreateClient()
	if err != nil {
		return err
	}

	log.Warnf("waiting for install to be ready, if this is the first time then it will take a while to download images")

	return kube.WaitForAllDeploymentsToBeReady(client, ns, 30*time.Minute)

}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
