package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/runner"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	JenkinsCookieName    = "JSESSIONID"
	JenkinsVersionHeader = "X-Jenkins"
)

var JenkinsReferenceVersion = semver.Version{Major: 2, Minor: 140, Patch: 0}

var (
	create_jenkins_user_long = templates.LongDesc(`
		Creates a new user and API Token for the current Jenkins server
`)

	create_jenkins_user_example = templates.Examples(`
		# Add a new API Token for a user for the current Jenkins server
        # prompting the user to find and enter the API Token
		jx create jenkins token someUserName

		# Add a new API Token for a user for the current Jenkins server
 		# using browser automation to login to the Git server
		# with the username an password to find the API Token
		jx create jenkins token -p somePassword someUserName	
	`)
)

// CreateJenkinsUserOptions the command line options for the command
type CreateJenkinsUserOptions struct {
	CreateOptions

	ServerFlags ServerFlags
	Username    string
	Password    string
	ApiToken    string
	BearerToken string
	Timeout     string
	UseBrowser  bool
}

// NewCmdCreateJenkinsUser creates a command
func NewCmdCreateJenkinsUser(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateJenkinsUserOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "token [username]",
		Short:   "Adds a new username and API token for a Jenkins server",
		Aliases: []string{"api-token"},
		Long:    create_jenkins_user_long,
		Example: create_jenkins_user_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	options.ServerFlags.addGitServerFlags(cmd)
	cmd.Flags().StringVarP(&options.ApiToken, "api-token", "t", "", "The API Token for the user")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "The User password to try automatically create a new API Token")
	cmd.Flags().StringVarP(&options.Timeout, "timeout", "", "", "The timeout if using browser automation to generate the API token (by passing username and password)")
	cmd.Flags().BoolVarP(&options.UseBrowser, "browser", "", false, "Use a Chrome browser to automatically find the API token if the user and password are known")

	return cmd
}

// Run implements the command
func (o *CreateJenkinsUserOptions) Run() error {
	args := o.Args
	if len(args) > 0 {
		o.Username = args[0]
	}
	if len(args) > 1 {
		o.ApiToken = args[1]
	}
	kubeClient, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return fmt.Errorf("error connecting to Kubernetes cluster: %v", err)
	}

	authConfigSvc, err := o.CreateJenkinsAuthConfigService(kubeClient, ns)
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	var server *auth.AuthServer
	if o.ServerFlags.IsEmpty() {
		url := ""
		url, err = o.findService(kube.ServiceJenkins)
		if err != nil {
			return err
		}
		server = config.GetOrCreateServer(url)
	} else {
		server, err = o.findServer(config, &o.ServerFlags, "jenkins server", "Try installing one via: jx create team", false)
		if err != nil {
			return err
		}
	}

	// TODO add the API thingy...
	if o.Username == "" {
		return fmt.Errorf("No Username specified")
	}

	userAuth := config.GetOrCreateUserAuth(server.URL, o.Username)
	if o.ApiToken != "" {
		userAuth.ApiToken = o.ApiToken
	}

	if o.BearerToken != "" {
		userAuth.BearerToken = o.BearerToken
	}

	if o.Password != "" {
		userAuth.Password = o.Password
	}

	tokenUrl := jenkins.JenkinsTokenURL(server.URL)
	if o.Verbose {
		log.Infof("Using url %s\n", tokenUrl)
	}
	if userAuth.IsInvalid() && o.Password != "" && o.UseBrowser {
		newTokenUrl := jenkins.JenkinsNewTokenURL(server.URL)
		err := o.tryFindAPITokenFromBrowser(tokenUrl, newTokenUrl, userAuth)
		if err != nil {
			log.Warnf("Unable to automatically find API token with chromedp using URL %s\n", tokenUrl)
			log.Warnf("Error: %v\n", err)
		}
	}

	if userAuth.IsInvalid() {
		f := func(username string) error {
			jenkins.PrintGetTokenFromURL(o.Out, tokenUrl)
			log.Infof("Then COPY the token and enter in into the form below:\n\n")
			return nil
		}

		err = config.EditUserAuth("Jenkins", userAuth, o.Username, false, o.BatchMode, f, o.In, o.Out, o.Err)
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

	// now lets create a secret for it so we can perform incluster interactions with Jenkins
	s, err := kubeClient.CoreV1().Secrets(o.currentNamespace).Get(kube.SecretJenkins, metav1.GetOptions{})
	if err != nil {
		return err
	}
	s.Data[kube.JenkinsAdminApiToken] = []byte(userAuth.ApiToken)
	s.Data[kube.JenkinsBearTokenField] = []byte(userAuth.BearerToken)
	s.Data[kube.JenkinsAdminUserField] = []byte(userAuth.Username)
	_, err = kubeClient.CoreV1().Secrets(o.currentNamespace).Update(s)
	if err != nil {
		return err
	}

	log.Infof("Created user %s API Token for Jenkins server %s at %s\n",
		util.ColorInfo(o.Username), util.ColorInfo(server.Name), util.ColorInfo(server.URL))
	return nil
}

func (o *CreateJenkinsUserOptions) tryFindAPITokenFromBrowser(tokenUrl string, newTokenUrl string, userAuth *auth.UserAuth) error {
	var ctx context.Context
	var cancel context.CancelFunc
	if o.Timeout != "" {
		duration, err := time.ParseDuration(o.Timeout)
		if err != nil {
			return errors.Wrap(err, "parsing the timeout value")
		}
		ctx, cancel = context.WithTimeout(context.Background(), duration)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	userDataDir, err := ioutil.TempDir("/tmp", "jx-login-chrome-userdata-dir")
	if err != nil {
		return errors.Wrap(err, "creating the chrome user data dir")
	}
	defer util.DestroyFile(userDataDir)
	netLogFile := filepath.Join(userDataDir, "net-logs.json")

	c, err := o.createChromeClientWithNetLog(ctx, userDataDir, netLogFile)
	if err != nil {
		return errors.Wrap(err, "creating the chrome client")
	}

	err = c.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(tokenUrl),
	})
	if err != nil {
		return errors.Wrapf(err, "navigating to token URL '%s'", tokenUrl)
	}

	nodeSlice := []*cdp.Node{}
	err = c.Run(ctx, chromedp.Nodes("//input", &nodeSlice))
	if err != nil {
		return errors.Wrap(err, "serching the login form")
	}

	login := false
	userNameInputName := "j_username"
	passwordInputSelector := "//input[@name='j_password']"
	for _, node := range nodeSlice {
		name := node.AttributeValue("name")
		if name == userNameInputName {
			login = true
		}
	}

	headerId := "header"
	if login {
		log.Infoln("Generating the API token...")
		err = c.Run(ctx, chromedp.Tasks{
			chromedp.WaitVisible(userNameInputName, chromedp.ByID),
			chromedp.SendKeys(userNameInputName, userAuth.Username, chromedp.ByID),
			chromedp.SendKeys(passwordInputSelector, o.Password+"\n"),
			chromedp.WaitVisible(headerId, chromedp.ByID),
			chromedp.ActionFunc(func(ctxt context.Context, h cdp.Executor) error {
				cookies, err := network.GetCookies().Do(ctxt, h)
				if err != nil {
					return err
				}
				for _, cookie := range cookies {
					if strings.HasPrefix(cookie.Name, JenkinsCookieName) {
						jenkinsCookie := cookie.Name + "=" + cookie.Value
						token, err := o.generateNewApiToken(newTokenUrl, jenkinsCookie)
						if err != nil {
							return errors.Wrap(err, "generating the API token")
						}
						if token != "" {
							userAuth.ApiToken = token
							return nil
						} else {
							return errors.New("received an empty API token")
						}
					}
				}

				return errors.New("no Jenkins cookie found after login")
			}),
		})
		if err != nil {
			return errors.Wrap(err, "generating the API token")
		}
	}

	err = c.Shutdown(ctx)
	if err != nil {
		return errors.Wrap(err, "shutting down the chrome client")
	}

	return nil
}

func (o *CommonOptions) generateNewApiToken(newTokenUrl string, cookie string) (string, error) {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodPost, newTokenUrl, nil)
	if err != nil {
		return "", errors.Wrap(err, "building request to generate the API token")
	}
	parts := strings.Split(cookie, "=")
	if len(parts) != 2 {
		return "", errors.Wrap(err, "building jenkins cookie")
	}
	jenkinsCookie := http.Cookie{Name: parts[0], Value: parts[1]}
	req.AddCookie(&jenkinsCookie)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "execute generate API token request")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading API token from response body")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("generate API token status code: %d, error: %s", resp.StatusCode, string(body))
	}

	type TokenData struct {
		TokenName  string `json:"tokenName"`
		TokenUuid  string `json:"tokenUuid"`
		TokenValue string `json:"tokenValue"`
	}

	type TokenResponse struct {
		Status string    `json:"status"`
		Data   TokenData `json:"data"`
	}
	tokenResponse := &TokenResponse{}
	if err := json.Unmarshal(body, tokenResponse); err != nil {
		return "", errors.Wrap(err, "parsing the API token from response")
	}
	return tokenResponse.Data.TokenValue, nil
}

func (o *CommonOptions) extractJenkinsCookie(text string) string {
	start := strings.Index(text, JenkinsCookieName)
	if start < 0 {
		return ""
	}
	end := -1
	for i, ch := range text[start:] {
		if ch == '"' {
			end = start + i
			break
		}
	}
	if end < 0 {
		return ""
	}
	return text[start:end]
}

// lets try use the users browser to find the API token
func (o *CommonOptions) createChromeClient(ctxt context.Context) (*chromedp.CDP, error) {
	if o.Headless {
		options := func(m map[string]interface{}) error {
			m["remote-debugging-port"] = 9222
			m["no-sandbox"] = true
			m["headless"] = true
			return nil
		}

		return chromedp.New(ctxt, chromedp.WithRunnerOptions(runner.CommandLineOption(options)))
	}
	return chromedp.New(ctxt)
}

func (o *CommonOptions) createChromeClientWithNetLog(ctx context.Context, userDataDir string, netLogFile string) (*chromedp.CDP, error) {
	options := func(m map[string]interface{}) error {
		if o.Headless {
			m["remote-debugging-port"] = 9222
			m["no-sandbox"] = true
			m["headless"] = true
		}
		m["user-data-dir"] = userDataDir
		m["log-net-log"] = netLogFile
		m["net-log-capture-mode"] = "IncludeCookiesAndCredentials"
		m["v"] = 1
		return nil
	}

	logger := func(string, ...interface{}) {
		return
	}
	return chromedp.New(ctx,
		chromedp.WithRunnerOptions(runner.CommandLineOption(options)),
		chromedp.WithLog(logger))
}

func (o *CommonOptions) captureScreenshot(ctxt context.Context, c *chromedp.CDP, screenshotFile string, selector interface{}, options ...chromedp.QueryOption) error {
	log.Infoln("Creating a screenshot...")

	var picture []byte
	err := c.Run(ctxt, chromedp.Tasks{
		chromedp.Sleep(2 * time.Second),
		chromedp.Screenshot(selector, &picture, options...),
	})
	if err != nil {
		return err
	}
	log.Infoln("Saving a screenshot...")

	err = ioutil.WriteFile(screenshotFile, picture, util.DefaultWritePermissions)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Infof("Saved screenshot: %s\n", util.ColorInfo(screenshotFile))
	return err
}

func (o *CommonOptions) createChromeDPLogger() (func(string, ...interface{}), error) {
	var logger func(string, ...interface{})
	if o.Verbose {
		logger = func(message string, args ...interface{}) {
			log.Infof(message+"\n", args...)
		}
	} else {
		file, err := ioutil.TempFile("", "jx-browser")
		if err != nil {
			return logger, err
		}
		writer := bufio.NewWriter(file)
		log.Infof("Chrome debugging logs written to: %s\n", util.ColorInfo(file.Name()))

		logger = func(message string, args ...interface{}) {
			fmt.Fprintf(writer, message+"\n", args...)
		}
	}
	return logger, nil
}
