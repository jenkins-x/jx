package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"os"

	"net/http"

	"encoding/base64"

	"fmt"
	"io/ioutil"

	"bytes"

	"strings"

	"encoding/json"

	"errors"

	"path/filepath"

	"os/exec"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/src-d/go-git.v4"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type InstallOptions struct {
	Factory cmdutil.Factory
	Out     io.Writer
	Err     io.Writer
	Flags   InstallFlags
}

type InstallFlags struct {
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
		Factory: f,
		Out:     out,
		Err:     errOut,
	}

	cmd := &cobra.Command{
		Use:     "install [flags]",
		Short:   "Install Jenkins-X",
		Long:    instalLong,
		Example: instalExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := RunInstall(f, out, errOut, cmd, args, options)
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"list", "ps"},
	}

	cmd.Flags().StringP("git-provider", "", "GitHub", "Git provider, used to create tokens if not provided.  Supported providers: [GitHub]")
	cmd.Flags().StringP("git-token", "t", "", "Git token used to clone and tag releases, typically using a bot user account.  For GitHub use a personal access token with 'public_repo' scope, see https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line")
	cmd.Flags().StringP("git-username", "u", "", "Git username used to tag releases in pipelines, typically this is a bot user")
	cmd.Flags().StringP("git-password", "p", "", "Git username if a Personal Access Token should be created")
	cmd.Flags().StringP("domain", "d", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	cmd.Flags().StringP("kubernetes-provider", "k", "minikube", "Service providing the kubernetes cluster.  Supported providers: [minikube,gke,thunder]")
	cmd.Flags().StringP("cloud-environment-repo", "c", "https://github.com/jenkins-x/cloud-environments", "Cloud Environments git repo")
	return cmd
}

// RunInstall implements the generic Install command
func RunInstall(f cmdutil.Factory, out, errOut io.Writer, cmd *cobra.Command, args []string, options *InstallOptions) error {
	flags := InstallFlags{
		Domain:             cmd.Flags().Lookup("domain").Value.String(),
		GitProvider:        cmd.Flags().Lookup("git-provider").Value.String(),
		GitToken:           cmd.Flags().Lookup("git-token").Value.String(),
		GitUser:            cmd.Flags().Lookup("git-username").Value.String(),
		GitPass:            cmd.Flags().Lookup("git-password").Value.String(),
		KubernetesProvider: cmd.Flags().Lookup("kubernetes-provider").Value.String(),
		CloudEnvRepository: cmd.Flags().Lookup("cloud-environment-repo").Value.String(),
	}
	options.Flags = flags

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
	makefileDir := filepath.Join(wrkDir, fmt.Sprintf("env-%s", strings.ToLower(options.Flags.KubernetesProvider)))

	err = ioutil.WriteFile(filepath.Join(makefileDir, "secrets.yaml"), []byte(secrets), 0644)
	if err != nil {
		return err
	}

	makefile := exec.Command("make", "install")

	makefile.Dir = makefileDir
	makefile.Stdout = out
	makefile.Stderr = errOut
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
		URL:           o.Flags.CloudEnvRepository,
		ReferenceName: "refs/heads/master",
		SingleBranch:  true,
		Progress:      o.Out,
	})
	if err != nil {
		if strings.Contains(err.Error(), "repository already exists") {
			log.Infof("Jenkins-X cloud environments repository already exists, check for changes? y/n: ")
			if log.AskForConfirmation(false) {

				r, err := git.PlainOpen(wrkDir)
				if err != nil {
					return err
				}

				// Get the working directory for the repository
				w, err := r.Worktree()
				if err != nil {
					return err
				}

				// Pull the latest changes from the origin remote and merge into the current branch
				err = w.Pull(&git.PullOptions{RemoteName: "origin"})
				if err != nil && !strings.Contains(err.Error(), "already up-to-date") {
					return err
				}

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

	username := o.Flags.GitUser
	if username == "" {
		if os.Getenv(JX_GIT_USER) != "" {
			username = os.Getenv(JX_GIT_USER)
		} else {
			log.Info("Git username to tag releases: ")
			_, err := fmt.Scanln(&username)
			if err != nil {
				errors.New(fmt.Sprintf("error reading username: %v", err))
			}
		}
	}

	// first check git-token flag
	if o.Flags.GitToken != "" {
		return username, o.Flags.GitToken, nil
	}

	// second check for an environment variable
	if os.Getenv(JX_GIT_TOKEN) != "" {
		return username, os.Getenv(JX_GIT_TOKEN), nil
	}

	// third if github provider request a new personal access token
	log.Warn("No flag --git-token or JX_GIT_TOKEN environment variable found, this is required so Jenkins-X can setup the secrets to clone and tag your releases\n")

	if o.Flags.GitProvider == "GitHub" {
		//fmt.Print("Would you like to create a new GitHub personal access token now? (y):")
		log.Info("Would you like to create a new GitHub personal access token now? y/n: ")

		if log.AskForConfirmation(false) {
			return o.createGitHubPersonalAccessToken(username)
		} else {
			os.Exit(-1)
		}
	}

	return "", "", nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (o *InstallOptions) createGitHubPersonalAccessToken(username string) (string, string, error) {

	password := o.Flags.GitPass
	if password == "" {
		if os.Getenv(JX_GIT_PASSWORD) != "" {
			password = os.Getenv(JX_GIT_PASSWORD)
		} else {
			log.Infof("GitHub password for user/bot [%s]: ", username)
			b, err := terminal.ReadPassword(0)
			log.Error("\n")
			if err != nil {
				errors.New(fmt.Sprintf("error reading password: %v", err))
			}
			password = string(b)
		}
	}

	client := &http.Client{}

	b := bytes.NewBufferString("{\"scopes\":[\"public_repo\"],\"note\":\"jx-bot\"}")

	req, err := http.NewRequest("POST", "https://api.github.com/authorizations", b)
	req.Header.Add("Authorization", "Basic "+basicAuth(username, password))

	resp, err := client.Do(req)
	if err != nil {
		errors.New(fmt.Sprintf("error creating github authorization: %v", err))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errors.New(fmt.Sprintf("error reading create authorization response: %v", err))
	}

	if strings.Contains(string(body), "already_exists") {
		log.Error("A jx-bot personal access token already exists, check here: https://github.com/settings/tokens\n")
		log.Info("Reuse this with the --git-token flag or delete from GitHub and try again.\n")
		os.Exit(-1)
	}

	var dat map[string]interface{}

	if dat == nil{
		log.Error("Not a valid user\n")
		log.Info("Ensure the user is valid on GitHub.\n")
		os.Exit(-1)
	}
	err = json.Unmarshal(body, &dat)
	if err != nil {
		errors.New(fmt.Sprintf("error unmarshalling authorization response: %v", err))
	}


	token := dat["token"].(string)
	log.Successf("Your new GitHub personal access token: %s", token)

	return username, token, nil
}
