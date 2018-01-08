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

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"golang.org/x/crypto/ssh/terminal"
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
	Domain      string
	GitProvider string
	GitToken    string
	GHUser      string
	GHPass      string
}

const (
	JX_GIT_TOKEN = "JX_GIT_TOKEN"
)

var (
	instalLong = templates.LongDesc(`
		Installs the Jenkins X platform on a Kubernetes cluster

`)

	instalExample = templates.Examples(`
		# Default installer
		jx install

		# Install with custom domain
		jx install -d jenkinsx.io

		# Install with a GitHub personal access token
		jx install -t 9fdbd2d070cd81eb12bca87861bcd850
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

	cmd.Flags().StringP("git-provider", "g", "GitHub", "Git provider, used to create")
	cmd.Flags().StringP("git-token", "t", "", "Git token used to clone and tag releases, typically using a bot user account.  For GitHub use a personal access token with 'public_repo' scope, see https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line")
	cmd.Flags().StringP("domain", "d", "", "Domain to expose ingress endpoints.  Example: jenkinsx.io")
	cmd.Flags().StringP("gh-username", "u", "", "GitHub username if a Personal Access Token should be created")
	cmd.Flags().StringP("gh-password", "p", "", "GitHub username if a Personal Access Token should be created")
	return cmd
}

// RunInstall implements the generic Install command
func RunInstall(f cmdutil.Factory, out, errOut io.Writer, cmd *cobra.Command, args []string, options *InstallOptions) error {
	flags := InstallFlags{
		Domain:      cmd.Flags().Lookup("domain").Value.String(),
		GitProvider: cmd.Flags().Lookup("git-provider").Value.String(),
		GitToken:    cmd.Flags().Lookup("git-token").Value.String(),
		GHUser:      cmd.Flags().Lookup("gh-username").Value.String(),
		GHPass:      cmd.Flags().Lookup("gh-password").Value.String(),
	}
	options.Flags = flags

	_, err := options.getGitToken()
	if err != nil {
		return err
	}

	// clone the environments repo

	// run  helm install setting the token and domain values

	return nil
}

// returns the Git Token that should be used by Jenkins-X to setup credentials to clone repos and creates a secret for pipelines to tag a release
func (o *InstallOptions) getGitToken() (string, error) {

	// first check git-token flag
	if o.Flags.GitToken != "" {
		return o.Flags.GitToken, nil
	}

	// second check for an environment variable
	if os.Getenv(JX_GIT_TOKEN) != "" {
		return os.Getenv(JX_GIT_TOKEN), nil
	}

	// third if github provider request a new personal access token
	log.Warn("No flag --git-token or JX_GIT_TOKEN environment variable found, this is required so Jenkins-X can setup the secrets to clone and tag your releases\n")

	if o.Flags.GitProvider == "GitHub" {
		//fmt.Print("Would you like to create a new GitHub personal access token now? (y):")
		log.Info("Would you like to create a new GitHub personal access token now? y/n: ")

		if log.AskForConfirmation(false) {
			return o.createGitHubPersonalAccessToken()
		}
	}

	return "", nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (o *InstallOptions) createGitHubPersonalAccessToken() (string, error) {
	username := o.Flags.GHUser
	if username == "" {
		log.Info("GitHub username: ")
		_, err := fmt.Scanln(&username)
		if err != nil {
			errors.New(fmt.Sprintf("error reading username: %v", err))
		}
	}

	password := o.Flags.GHPass
	if password == "" {
		log.Info("GitHub password: ")
		b, err := terminal.ReadPassword(0)
		log.Error("\n")
		if err != nil {
			errors.New(fmt.Sprintf("error reading password: %v", err))
		}
		password = string(b)
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
	err = json.Unmarshal(body, &dat)
	if err != nil {
		errors.New(fmt.Sprintf("error unmarshalling authorization response: %v", err))
	}

	token := dat["token"].(string)
	log.Successf("Your new GitHub personal access token: %s", token)

	return token, nil
}
