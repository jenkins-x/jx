package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/nodes"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	jenkinsGitCredentialsSecretKey = "credentials"
)

var (
	create_git_token_long = templates.LongDesc(`
		Creates a new API Token for a user on a Git Server
`)

	create_git_token_example = templates.Examples(`
		# Add a new API Token for a user for the local git server
        # prompting the user to find and enter the API Token
		jx create git token -n local someUserName

		# Add a new API Token for a user for the local git server 
 		# using browser automation to login to the git server
		# with the username an password to find the API Token
		jx create git token -n local -p somePassword someUserName	
	`)
)

// CreateGitTokenOptions the command line options for the command
type CreateGitTokenOptions struct {
	CreateOptions

	ServerFlags ServerFlags
	Username    string
	Password    string
	ApiToken    string
	Timeout     string
}

// NewCmdCreateGitToken creates a command
func NewCmdCreateGitToken(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateGitTokenOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "token [username]",
		Short:   "Adds a new API token for a user on a git server",
		Aliases: []string{"api-token"},
		Long:    create_git_token_long,
		Example: create_git_token_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	options.ServerFlags.addGitServerFlags(cmd)
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

	server, err := o.findGitServer(config, &o.ServerFlags)
	if err != nil {
		return err
	}
	err = o.ensureGitServiceCRD(server)
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

		err = config.EditUserAuth(server.Label(), userAuth, o.Username, false, o.BatchMode, f)
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

	err = o.updateGitCredentialsSecret(server, userAuth)
	if err != nil {
		log.Warnf("Failed to update jenkins git credentials secret: %v\n", err)
	}

	_, err = o.updatePipelineGitCredentialsSecret(server, userAuth)
	if err != nil {
		log.Warnf("Failed to update Jenkins X pipeline git credentials secret: %v\n", err)
	}

	log.Infof("Created user %s API Token for git server %s at %s\n",
		util.ColorInfo(o.Username), util.ColorInfo(server.Name), util.ColorInfo(server.URL))

	return nil
}

// lets try use the users browser to find the API token
func (o *CreateGitTokenOptions) tryFindAPITokenFromBrowser(tokenUrl string, userAuth *auth.UserAuth) error {
	var ctxt context.Context
	var cancel context.CancelFunc
	if o.Timeout != "" {
		duration, err := time.ParseDuration(o.Timeout)
		if err != nil {
			return err
		}
		ctxt, cancel = context.WithTimeout(context.Background(), duration)
	} else {
		ctxt, cancel = context.WithCancel(context.Background())
	}
	defer cancel()
	log.Infof("Trying to generate an API token for user: %s\n", util.ColorInfo(userAuth.Username))

	c, err := o.createChromeClient(ctxt)
	if err != nil {
		return err
	}

	err = c.Run(ctxt, chromedp.Tasks{
		chromedp.Navigate(tokenUrl),
	})
	if err != nil {
		return err
	}

	nodeSlice := []*cdp.Node{}
	err = c.Run(ctxt, chromedp.Nodes("//input", &nodeSlice))
	if err != nil {
		return err
	}

	login := false
	for _, node := range nodeSlice {
		name := node.AttributeValue("name")
		if name == "user_name" {
			login = true
		}
	}

	if login {
		o.captureScreenshot(ctxt, c, "screenshot-git-login.png", "//div")

		log.Infof("logging in\n")
		err = c.Run(ctxt, chromedp.Tasks{
			chromedp.WaitVisible("user_name", chromedp.ByID),
			chromedp.SendKeys("user_name", userAuth.Username, chromedp.ByID),
			chromedp.SendKeys("password", o.Password+"\n", chromedp.ByID),
		})
		if err != nil {
			return err
		}
	}

	o.captureScreenshot(ctxt, c, "screenshot-git-api-token.png", "//div")

	log.Infoln("Generating new token")

	tokenId := "jx-" + string(uuid.NewUUID())
	generateNewTokenButtonSelector := "//div[normalize-space(text())='Generate New Token']"

	tokenResultDivSelector := "//div[@class='ui info message']/p"
	err = c.Run(ctxt, chromedp.Tasks{
		chromedp.WaitVisible(generateNewTokenButtonSelector),
		chromedp.Click(generateNewTokenButtonSelector),
		chromedp.WaitVisible("name", chromedp.ByID),
		chromedp.SendKeys("name", tokenId+"\n", chromedp.ByID),
		chromedp.WaitVisible(tokenResultDivSelector),
		chromedp.Nodes(tokenResultDivSelector, &nodeSlice),
	})
	if err != nil {
		return err
	}
	token := ""
	for _, node := range nodeSlice {
		text := nodes.NodeText(node)
		if text != "" && token == "" {
			token = text
			break
		}
	}
	log.Infoln("Found API Token")
	if token != "" {
		userAuth.ApiToken = token
	}

	err = c.Shutdown(ctxt)
	if err != nil {
		return err
	}
	return nil
}

func (o *CreateGitTokenOptions) updateGitCredentialsSecret(server *auth.AuthServer, userAuth *auth.UserAuth) error {
	client, ns, err := o.KubeClient()
	if err != nil {
		return err
	}
	options := metav1.GetOptions{}
	secret, err := client.CoreV1().Secrets(ns).Get(kube.SecretJenkinsGitCredentials, options)
	if err != nil {
		// lets try the real dev namespace if we are in a different environment
		devNs, _, err := kube.GetDevNamespace(client, ns)
		if err != nil {
			return err
		}
		secret, err = client.CoreV1().Secrets(devNs).Get(kube.SecretJenkinsGitCredentials, options)
		if err != nil {
			return err
		}
	}
	text := ""
	data := secret.Data[jenkinsGitCredentialsSecretKey]
	if data != nil {
		text = string(data)
	}
	lines := strings.Split(text, "\n")
	u, err := url.Parse(server.URL)
	if err != nil {
		return fmt.Errorf("Failed to parse server URL %s due to: %s", server.URL, err)
	}

	found := false
	prefix := u.Scheme + "://"
	host := u.Host
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) && strings.HasSuffix(line, host) {
			found = true
		}
	}
	if !found {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += prefix + userAuth.Username + ":" + userAuth.ApiToken + "@" + host
		secret.Data[jenkinsGitCredentialsSecretKey] = []byte(text)
		_, err = client.CoreV1().Secrets(ns).Update(secret)
		if err != nil {
			return fmt.Errorf("Failed to update secret %s due to %s", secret.Name, err)
		}
	}
	return nil
}
